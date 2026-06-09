package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

const (
	sessionTableName              = "session_table"
	sessionMaxPartitionName       = "pmax"
	sessionPartitionCheckInterval = 24 * time.Hour
	sessionCleanupCheckInterval   = time.Hour
	dbOperationTimeout            = 30 * time.Second
	mysqlDateTimeLayout           = "2006-01-02 15:04:05"
	defaultPartitionDays          = 1
	defaultRetentionCleanupHour   = 2
	maxVerbosity                  = 5
)

var requiredSchemaTables = []string{
	"cgnat_table",
	"whitelist_table",
	"user_table",
	sessionTableName,
	"alarm_table",
}

type sessionPartition struct {
	Name     string
	LessThan time.Time
}

func normalizeConfig(cfg Config) Config {
	if cfg.PartitionDays <= 0 {
		cfg.PartitionDays = defaultPartitionDays
	}
	if cfg.Verbosity < 0 {
		cfg.Verbosity = 0
	}
	if cfg.Verbosity > maxVerbosity {
		cfg.Verbosity = maxVerbosity
	}
	return cfg
}

func validateConfig(cfg Config) error {
	if cfg.RetentionCleanupHour < 0 || cfg.RetentionCleanupHour > 23 {
		return fmt.Errorf("retention_cleanup_hour must be between 0 and 23")
	}
	if cfg.RetentionPeriod < 0 {
		return fmt.Errorf("retention_period cannot be negative")
	}
	return nil
}

func (c *Connection) log(level int, format string, args ...any) {
	if c == nil || c.cfg.Verbosity < level {
		return
	}
	log.Printf("[db] "+format, args...)
}

func (c *Connection) logError(format string, args ...any) {
	log.Printf("[db] ERROR: "+format, args...)
}

func (c *Connection) partitionLookahead() time.Duration {
	return time.Duration(c.cfg.PartitionDays) * 24 * time.Hour
}

func (c *Connection) retentionPeriod() time.Duration {
	return time.Duration(c.cfg.RetentionPeriod) * 24 * time.Hour
}

func (c *Connection) applySchema(ctx context.Context) error {
	missing, err := c.missingSchemaTables(ctx)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		c.log(2, "schema already present")
		return nil
	}

	if strings.TrimSpace(c.cfg.MySQLSchema) == "" {
		return fmt.Errorf("schema is missing tables %s and mysql_schema is empty", strings.Join(missing, ", "))
	}

	c.log(1, "schema missing tables: %s; applying %s", strings.Join(missing, ", "), c.cfg.MySQLSchema)

	schemaBytes, err := os.ReadFile(c.cfg.MySQLSchema)
	if err != nil {
		return fmt.Errorf("read schema file: %w", err)
	}

	statements := splitSQLStatements(string(schemaBytes))
	c.log(3, "executing %d schema statements", len(statements))
	for i, statement := range statements {
		if _, err := c.conn.ExecContext(ctx, statement); err != nil {
			if isIgnorableSchemaError(err) {
				c.log(4, "ignored idempotent schema error at statement %d: %v", i+1, err)
				continue
			}
			return fmt.Errorf("schema exec failed: %w", err)
		}
		c.log(5, "schema statement %d executed", i+1)
	}

	missing, err = c.missingSchemaTables(ctx)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("schema file applied but required tables are still missing: %s", strings.Join(missing, ", "))
	}
	c.log(1, "schema applied successfully")
	return nil
}

func (c *Connection) missingSchemaTables(ctx context.Context) ([]string, error) {
	rows, err := c.conn.QueryContext(ctx, `
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ?
	`, c.cfg.MySQLDatabase)
	if err != nil {
		return nil, fmt.Errorf("check schema tables: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]bool, len(requiredSchemaTables))
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("scan schema table: %w", err)
		}
		existing[table] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan schema tables: %w", err)
	}

	var missing []string
	for _, table := range requiredSchemaTables {
		if !existing[table] {
			missing = append(missing, table)
		}
	}
	return missing, nil
}

func (c *Connection) startPartitionManager() {
	c.log(2, "session partition manager started; interval=%s lookahead_days=%d", sessionPartitionCheckInterval, c.cfg.PartitionDays)

	go func() {
		ticker := time.NewTicker(sessionPartitionCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
				if err := c.syncSessionPartitions(ctx, time.Now().UTC()); err != nil {
					c.logError("session partition sync failed: %v", err)
				}
				cancel()
			case <-c.maintenanceStop:
				c.log(3, "session partition manager stopped")
				return
			}
		}
	}()
}

func (c *Connection) syncSessionPartitions(ctx context.Context, now time.Time) error {
	c.maintenanceMu.Lock()
	defer c.maintenanceMu.Unlock()

	c.log(4, "checking session partitions at %s", now.UTC().Format(mysqlDateTimeLayout))

	existing, err := c.existingSessionPartitions(ctx)
	if err != nil {
		return err
	}
	if !existing[sessionMaxPartitionName] {
		return fmt.Errorf("%s is missing %s partition", sessionTableName, sessionMaxPartitionName)
	}

	var missing []sessionPartition
	for _, partition := range c.desiredSessionPartitions(now) {
		if !existing[partition.Name] {
			missing = append(missing, partition)
		}
	}
	if len(missing) == 0 {
		c.log(2, "session partitions are current")
		return nil
	}

	c.log(2, "creating %d session partitions", len(missing))
	c.log(4, "missing session partitions: %s", sessionPartitionNames(missing))

	if err := c.reorganizeSessionMaxPartition(ctx, missing); err != nil {
		if isDuplicatePartitionError(err) {
			c.log(3, "session partition sync raced with an existing partition; skipping duplicate")
			return nil
		}
		return err
	}
	c.log(1, "created %d session partitions through %s", len(missing), missing[len(missing)-1].LessThan.Format(mysqlDateTimeLayout))
	return nil
}

func (c *Connection) existingSessionPartitions(ctx context.Context) (map[string]bool, error) {
	rows, err := c.conn.QueryContext(ctx, `
		SELECT PARTITION_NAME
		FROM INFORMATION_SCHEMA.PARTITIONS
		WHERE TABLE_SCHEMA = ?
		AND TABLE_NAME = ?
		AND PARTITION_NAME IS NOT NULL
	`, c.cfg.MySQLDatabase, sessionTableName)
	if err != nil {
		return nil, fmt.Errorf("check session partitions: %w", err)
	}
	defer rows.Close()

	partitions := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan session partition: %w", err)
		}
		partitions[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan session partitions: %w", err)
	}
	return partitions, nil
}

func (c *Connection) reorganizeSessionMaxPartition(ctx context.Context, partitions []sessionPartition) error {
	var b strings.Builder
	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdentifier(sessionTableName))
	b.WriteString(" REORGANIZE PARTITION ")
	b.WriteString(quoteIdentifier(sessionMaxPartitionName))
	b.WriteString(" INTO (")

	for i, partition := range partitions {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("PARTITION ")
		b.WriteString(quoteIdentifier(partition.Name))
		b.WriteString(" VALUES LESS THAN ('")
		b.WriteString(partition.LessThan.Format(mysqlDateTimeLayout))
		b.WriteString("')")
	}

	b.WriteString(", PARTITION ")
	b.WriteString(quoteIdentifier(sessionMaxPartitionName))
	b.WriteString(" VALUES LESS THAN (MAXVALUE))")

	if _, err := c.conn.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("reorganize session partitions: %w", err)
	}
	return nil
}

func (c *Connection) startRetentionCleanupManager() {
	if c.cfg.RetentionPeriod <= 0 {
		c.log(1, "session retention cleanup disabled; retention_period <= 0")
		return
	}

	c.log(2, "session retention cleanup manager started; interval=%s retention_days=%d cleanup_hour=%02d:00 UTC", sessionCleanupCheckInterval, c.cfg.RetentionPeriod, c.cfg.RetentionCleanupHour)

	go func() {
		ticker := time.NewTicker(sessionCleanupCheckInterval)
		defer ticker.Stop()

		var lastRunDate string
		runIfDue := func(now time.Time) {
			if !shouldRunRetentionCleanup(now, lastRunDate, c.cfg.RetentionCleanupHour) {
				c.log(5, "session retention cleanup not due at %s", now.UTC().Format(mysqlDateTimeLayout))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
			if err := c.cleanupExpiredSessions(ctx, now.UTC()); err != nil {
				c.logError("session retention cleanup failed: %v", err)
			} else {
				lastRunDate = now.UTC().Format("2006-01-02")
			}
			cancel()
		}

		runIfDue(time.Now().UTC())

		for {
			select {
			case now := <-ticker.C:
				runIfDue(now.UTC())
			case <-c.maintenanceStop:
				c.log(3, "session retention cleanup manager stopped")
				return
			}
		}
	}()
}

func shouldRunRetentionCleanup(now time.Time, lastRunDate string, cleanupHour int) bool {
	now = now.UTC()
	return now.Hour() == cleanupHour && now.Format("2006-01-02") != lastRunDate
}

func (c *Connection) cleanupExpiredSessions(ctx context.Context, now time.Time) error {
	c.maintenanceMu.Lock()
	defer c.maintenanceMu.Unlock()

	cutoff := now.UTC().Add(-c.retentionPeriod())
	c.log(1, "starting session retention cleanup; cutoff=%s", cutoff.Format(mysqlDateTimeLayout))

	dropped, err := c.dropExpiredSessionPartitions(ctx, cutoff)
	if err != nil {
		return err
	}

	deleted, err := c.deleteExpiredSessionRows(ctx, cutoff)
	if err != nil {
		return err
	}

	c.log(1, "session retention cleanup complete; dropped_partitions=%d deleted_rows=%d cutoff=%s", dropped, deleted, cutoff.Format(mysqlDateTimeLayout))
	return nil
}

func (c *Connection) dropExpiredSessionPartitions(ctx context.Context, cutoff time.Time) (int, error) {
	partitions, err := c.expiredSessionPartitionNames(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	if len(partitions) == 0 {
		c.log(2, "no expired session partitions to drop")
		return 0, nil
	}

	c.log(2, "dropping %d expired session partitions", len(partitions))
	c.log(3, "expired session partitions: %s", strings.Join(partitions, ", "))

	var b strings.Builder
	b.WriteString("ALTER TABLE ")
	b.WriteString(quoteIdentifier(sessionTableName))
	b.WriteString(" DROP PARTITION ")
	for i, partition := range partitions {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quoteIdentifier(partition))
	}

	if _, err := c.conn.ExecContext(ctx, b.String()); err != nil {
		if isMissingPartitionError(err) {
			c.log(3, "expired partition disappeared before drop completed")
			return 0, nil
		}
		return 0, fmt.Errorf("drop expired session partitions: %w", err)
	}
	return len(partitions), nil
}

func (c *Connection) expiredSessionPartitionNames(ctx context.Context, cutoff time.Time) ([]string, error) {
	rows, err := c.conn.QueryContext(ctx, `
		SELECT PARTITION_NAME
		FROM INFORMATION_SCHEMA.PARTITIONS
		WHERE TABLE_SCHEMA = ?
		AND TABLE_NAME = ?
		AND PARTITION_NAME IS NOT NULL
		AND PARTITION_NAME <> ?
	`, c.cfg.MySQLDatabase, sessionTableName, sessionMaxPartitionName)
	if err != nil {
		return nil, fmt.Errorf("check expired session partitions: %w", err)
	}
	defer rows.Close()

	var expired []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan expired session partition: %w", err)
		}

		lessThan, ok := sessionPartitionLessThanFromName(name)
		if !ok {
			c.log(4, "ignoring session partition with unexpected name %q", name)
			continue
		}
		if !lessThan.After(cutoff.UTC()) {
			expired = append(expired, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan expired session partitions: %w", err)
	}

	sort.Slice(expired, func(i, j int) bool {
		return expired[i] < expired[j]
	})
	return expired, nil
}

func (c *Connection) deleteExpiredSessionRows(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := c.conn.ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM %s WHERE %s < ?", quoteIdentifier(sessionTableName), quoteIdentifier("end_time")),
		cutoff.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired session rows: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.log(4, "could not read expired session row count: %v", err)
		return 0, nil
	}
	return rowsAffected, nil
}

func (c *Connection) desiredSessionPartitions(now time.Time) []sessionPartition {
	now = now.UTC()
	start := now.Truncate(time.Hour)
	horizon := ceilHour(now.Add(c.partitionLookahead()))

	partitions := make([]sessionPartition, 0, int(horizon.Sub(start)/time.Hour))
	for t := start; t.Before(horizon); t = t.Add(time.Hour) {
		partitions = append(partitions, sessionPartition{
			Name:     sessionPartitionName(t),
			LessThan: t.Add(time.Hour),
		})
	}
	return partitions
}

func sessionPartitionNames(partitions []sessionPartition) string {
	names := make([]string, 0, len(partitions))
	for _, partition := range partitions {
		names = append(names, partition.Name)
	}
	return strings.Join(names, ", ")
}

func sessionPartitionName(t time.Time) string {
	return "p" + t.UTC().Format("2006010215")
}

func sessionPartitionLessThanFromName(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, "p") {
		return time.Time{}, false
	}

	start, err := time.ParseInLocation("2006010215", strings.TrimPrefix(name, "p"), time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return start.Add(time.Hour), true
}

func ceilHour(t time.Time) time.Time {
	truncated := t.Truncate(time.Hour)
	if truncated.Equal(t) {
		return t
	}
	return truncated.Add(time.Hour)
}

func splitSQLStatements(sqlText string) []string {
	var statements []string
	var current strings.Builder
	var inSingleQuote, inDoubleQuote, inBacktick bool
	var previous rune

	for _, r := range sqlText {
		if r == ';' && !inSingleQuote && !inDoubleQuote && !inBacktick {
			statement := strings.TrimSpace(current.String())
			if statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
			previous = 0
			continue
		}

		current.WriteRune(r)

		switch r {
		case '\'':
			if !inDoubleQuote && !inBacktick && previous != '\\' {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote && !inBacktick && previous != '\\' {
				inDoubleQuote = !inDoubleQuote
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
		}
		previous = r
	}

	statement := strings.TrimSpace(current.String())
	if statement != "" {
		statements = append(statements, statement)
	}
	return statements
}

func isIgnorableSchemaError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1061, 1826:
			return true
		}
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key name") ||
		strings.Contains(message, "duplicate foreign key")
}

func isDuplicatePartitionError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1517:
			return true
		}
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate partition")
}

func isMissingPartitionError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1507:
			return true
		}
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "partition") &&
		(strings.Contains(message, "doesn't exist") || strings.Contains(message, "does not exist"))
}

func mysqlDSN(cfg *Config, database string, multiStatements bool) string {
	mysqlCfg := mysqlDriver.NewConfig()
	mysqlCfg.User = cfg.MySQLUser
	mysqlCfg.Passwd = cfg.MySQLPassword
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = fmt.Sprintf("%s:%d", cfg.MySQLHost, cfg.MySQLPort)
	mysqlCfg.DBName = database
	mysqlCfg.ParseTime = true
	mysqlCfg.MultiStatements = multiStatements

	return mysqlCfg.FormatDSN()
}

func quoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}
