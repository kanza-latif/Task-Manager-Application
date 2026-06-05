package mq

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskmanager/internal/db"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

// BinlogListener watches MySQL binary log for changes in
// cgnat_table and whitelist_table, then publishes change events to RabbitMQ queues
type BinlogListener struct {
	mqClient *Client
	dbCfg    db.Config
	canal    *canal.Canal
}

// NewBinlogListener creates a new binlog listener.
func NewBinlogListener(mqClient *Client, dbCfg db.Config) *BinlogListener {
	return &BinlogListener{
		mqClient: mqClient,
		dbCfg:    dbCfg,
	}
}

// Start begins listening to the MySQL binary log.
// Blocks until any user interrupt is received
func (b *BinlogListener) Start(dbConn *sql.DB) error {
	// Get current binlog position to start from now (not replay history)
	pos, err := getCurrentBinlogPos(dbConn)
	if err != nil {
		return fmt.Errorf("failed to get binlog position: %w", err)
	}

	cfg := canal.NewDefaultConfig()
	cfg.Addr = fmt.Sprintf("%s:%s", b.dbCfg.Host, b.dbCfg.Port)
	cfg.User = b.dbCfg.User
	cfg.Password = b.dbCfg.Password
	cfg.Dump.TableDB = b.dbCfg.Name

	// Only watch these two tables
	cfg.IncludeTableRegex = []string{
		fmt.Sprintf("%s\\.cgnat_table", b.dbCfg.Name),
		fmt.Sprintf("%s\\.whitelist_table", b.dbCfg.Name),
	}

	c, err := canal.NewCanal(cfg)
	if err != nil {
		return fmt.Errorf("failed to create canal: %w", err)
	}
	b.canal = c

	// Register event handler
	c.SetEventHandler(&binlogHandler{mqClient: b.mqClient})

	fmt.Println("Binary log listener started — watching cgnat_table and whitelist_table...")
	fmt.Println("   Any INSERT, UPDATE, or DELETE will be published to RabbitMQ.")
	fmt.Println("   Press Ctrl+C to stop.\n")

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stop
		cancel()
		c.Close()
	}()

	_ = ctx
	return c.RunFrom(pos)
}

// binlogHandler handles row change events from the canal.
type binlogHandler struct {
	mqClient *Client
	canal.DummyEventHandler
}

func (h *binlogHandler) OnRow(e *canal.RowsEvent) error {
	table := e.Table.Name
	action := ""

	switch e.Action {
	case canal.InsertAction:
		action = "insert"
	case canal.UpdateAction:
		action = "update"
	case canal.DeleteAction:
		action = "delete"
	default:
		return nil
	}

	switch table {
	case "cgnat_table":
		return h.handleCGNATChange(action, e)
	case "whitelist_table":
		return h.handleWhitelistChange(action, e)
	}
	return nil
}

func (h *binlogHandler) handleCGNATChange(action string, e *canal.RowsEvent) error {
	// For updates, e.Rows contains [old_row, new_row] pairs
	// For insert/delete, e.Rows contains the affected rows
	rows := e.Rows
	if action == "update" && len(rows) >= 2 {
		rows = rows[1:] // use new values for update
	}

	for _, row := range rows {
		if len(row) < 5 {
			continue
		}

		msg := CGNATMessage{
			ID:        toInt64(row[0]),
			PrivateIP: rawToIP(toBytes(row[1])),
			PublicIP:  rawToIP(toBytes(row[2])),
			StartPort: int(toInt64(row[3])),
			EndPort:   int(toInt64(row[4])),
		}

		if err := h.mqClient.PublishChangeEvent("cgnat_table", action, msg); err != nil {
			return err
		}
	}
	return nil
}

func (h *binlogHandler) handleWhitelistChange(action string, e *canal.RowsEvent) error {
	rows := e.Rows
	if action == "update" && len(rows) >= 2 {
		rows = rows[1:]
	}

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}

		msg := WhitelistMessage{
			ID:     toInt64(row[0]),
			MSISDN: fmt.Sprintf("%v", row[1]),
		}

		if err := h.mqClient.PublishChangeEvent("whitelist_table", action, msg); err != nil {
			return err
		}
	}
	return nil
}

func (h *binlogHandler) String() string {
	return "binlogHandler"
}

// getCurrentBinlogPos reads the current MySQL binlog position
// so the listener starts from NOW, not from the beginning of history.
func getCurrentBinlogPos(db *sql.DB) (mysql.Position, error) {
	var file string
	var position uint32
	var binlogDoDB, binlogIgnoreDB, executedGtidSet string

	row := db.QueryRow("SHOW MASTER STATUS")
	err := row.Scan(&file, &position, &binlogDoDB, &binlogIgnoreDB, &executedGtidSet)
	if err != nil {
		return mysql.Position{}, fmt.Errorf("SHOW MASTER STATUS failed — ensure binary logging is enabled on MySQL: %w", err)
	}

	return mysql.Position{Name: file, Pos: position}, nil
}

// type conversion helpers for binlog row values

func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case uint64:
		return int64(val)
	case uint32:
		return int64(val)
	}
	return 0
}

func toBytes(v interface{}) []byte {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	}
	return nil
}

// reuse rawToIP from cgnat_repository via a local copy
// to avoid import cycle between mq and repository packages
func rawToIP(b []byte) string {
	if len(b) == 4 {
		return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
	}
	if len(b) == 0 {
		return ""
	}

	// try as integer string
	var n uint64
	_, err := fmt.Sscanf(string(b), "%d", &n)
	if err == nil && n > 0 {
		return fmt.Sprintf("%d.%d.%d.%d",
			(n>>24)&0xFF,
			(n>>16)&0xFF,
			(n>>8)&0xFF,
			n&0xFF,
		)
	}
	return string(b)
}

var _ = replication.BinlogEvent{}
var _ = time.Now
