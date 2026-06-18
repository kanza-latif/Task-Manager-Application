package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Verbosity int
	SiteName  string
	NodeName  string

	// RabbitMQ
	RabbitMQHost string
	RabbitMQPort int

	RabbitMQUser     string
	RabbitMQPassword string

	RabbitMQVHost    string
	RabbitMQExchange string

	RabbitMQAdminUser     string
	RabbitMQAdminPassword string

	// MySQL
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

	// DB Workers
	DBWorkers   int
	DBQueueSize int
}

// LoadConfig parses key=value file
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer f.Close()

	cfg := &Config{
		PartitionDays:        1,
		RetentionCleanupHour: 2,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// remove inline comments
		if idx := strings.Index(val, "#"); idx != -1 {
			val = strings.TrimSpace(val[:idx])
		}

		cfg.apply(key, val)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	cfg.normalize()
	return cfg, nil
}

func (c *Config) apply(key, val string) {
	switch key {

	// rabbitmq
	case "rabbitmq_host":
		c.RabbitMQHost = val
	case "rabbitmq_port":
		c.RabbitMQPort = parseInt(val)
	case "rabbitmq_user":
		c.RabbitMQUser = val
	case "rabbitmq_password":
		c.RabbitMQPassword = val
	case "rabbitmq_admin_user":
		c.RabbitMQAdminUser = val
	case "rabbitmq_admin_password":
		c.RabbitMQAdminPassword = val
	case "rabbitmq_vhost":
		c.RabbitMQVHost = val
	case "rabbitmq_exchange":
		c.RabbitMQExchange = val

		// mysql
	case "mysql_host":
		c.MySQLHost = val
	case "mysql_port":
		c.MySQLPort = parseInt(val)
	case "mysql_user":
		c.MySQLUser = val
	case "mysql_password":
		c.MySQLPassword = val
	case "mysql_database":
		c.MySQLDatabase = val
	case "mysql_schema":
		c.MySQLSchema = val

	case "mysql_max_open_conns":
		c.MySQLMaxOpenConns = parseInt(val)
	case "mysql_max_idle_conns":
		c.MySQLMaxIdleConns = parseInt(val)
	case "mysql_conn_max_lifetime":
		c.MySQLConnMaxLifetime = parseInt(val)

	case "partition_days":
		c.PartitionDays = parseInt(val)
	case "retention_period":
		c.RetentionPeriod = parseInt(val)
	case "retention_cleanup_hour", "cleanup_hour":
		c.RetentionCleanupHour = parseInt(val)

	// database workers
	case "db_workers":
		c.DBWorkers = parseInt(val)
	case "db_queue_size":
		c.DBQueueSize = parseInt(val)

	case "verbosity":
		c.Verbosity = parseInt(val)
	case "site_name":
		c.SiteName = val
	case "node_name":
		c.NodeName = val
	}
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func (c *Config) normalize() {
	if c.PartitionDays <= 0 {
		c.PartitionDays = 1
	}
	if c.Verbosity < 0 {
		c.Verbosity = 0
	}
	if c.Verbosity > 5 {
		c.Verbosity = 5
	}
}
