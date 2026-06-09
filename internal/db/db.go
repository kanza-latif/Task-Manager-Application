package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"taskmanager/internal/repository"
)

type Config struct {
	MySQLHost     string
	MySQLPort     int
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string
	MySQLSchema   string

	MySQLMaxOpenConns    int
	MySQLMaxIdleConns    int
	MySQLConnMaxLifetime int

	PartitionDays        int
	RetentionPeriod      int
	RetentionCleanupHour int
	Verbosity            int

	// DB Workers
	DBWorkers   int
	DBQueueSize int
}

type Connection struct {
	cfg  Config
	conn *sql.DB

	maintenanceMu   sync.Mutex
	maintenanceStop chan struct{}
	stopOnce        sync.Once

	Alarm     *repository.AlarmRepository
	CGNAT     *repository.CGNATRepository
	Session   *repository.SessionRepository
	User      *repository.UserRepository
	Whitelist *repository.WhitelistRepository
}

var (
	GlobalConn *Connection
	initMu     sync.Mutex
)

func Init(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("database config cannot be nil")
	}

	initMu.Lock()
	defer initMu.Unlock()

	if GlobalConn != nil && GlobalConn.conn != nil {
		return nil
	}

	normalized := normalizeConfig(*cfg)
	if err := validateConfig(normalized); err != nil {
		return err
	}

	conn := &Connection{
		cfg:             normalized,
		maintenanceStop: make(chan struct{}),
	}

	if err := conn.connect(); err != nil {
		return err
	}

	GlobalConn = conn
	return nil
}

func (c *Connection) connect() (err error) {
	c.log(1, "initializing database connection")

	if err := EnsureDatabaseExists(&c.cfg); err != nil {
		return err
	}
	c.log(2, "database %q is present", c.cfg.MySQLDatabase)

	db, err := sql.Open("mysql", mysqlDSN(&c.cfg, c.cfg.MySQLDatabase, true))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = db.Close()
			c.conn = nil
		}
	}()

	db.SetMaxOpenConns(c.cfg.MySQLMaxOpenConns)
	db.SetMaxIdleConns(c.cfg.MySQLMaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(c.cfg.MySQLConnMaxLifetime) * time.Second)

	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}
	c.log(2, "connected to mysql at %s:%d", c.cfg.MySQLHost, c.cfg.MySQLPort)

	c.conn = db

	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	if err := c.applySchema(ctx); err != nil {
		return err
	}

	if err := c.syncSessionPartitions(ctx, time.Now().UTC()); err != nil {
		return err
	}

	if err := c.initRepositories(); err != nil {
		return err
	}
	c.log(2, "repository handles initialized")

	c.startPartitionManager()
	c.startRetentionCleanupManager()

	c.log(1, "database initialization complete")

	return nil
}

func EnsureDatabaseExists(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("database config cannot be nil")
	}
	if strings.TrimSpace(cfg.MySQLDatabase) == "" {
		return fmt.Errorf("mysql_database cannot be empty")
	}

	db, err := sql.Open("mysql", mysqlDSN(cfg, "", false))
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoteIdentifier(cfg.MySQLDatabase))

	_, err = db.Exec(query)

	return err
}

func Get() *sql.DB {
	if GlobalConn == nil || GlobalConn.conn == nil {
		panic("database not initialized")
	}

	return GlobalConn.conn
}

func Close() error {
	initMu.Lock()
	defer initMu.Unlock()

	if GlobalConn == nil {
		return nil
	}
	GlobalConn.stopMaintenance()

	if GlobalConn.conn == nil {
		GlobalConn = nil
		return nil
	}

	err := GlobalConn.conn.Close()
	GlobalConn = nil
	return err
}

func (c *Connection) stopMaintenance() {
	if c.maintenanceStop == nil {
		return
	}
	c.stopOnce.Do(func() {
		close(c.maintenanceStop)
	})
}
