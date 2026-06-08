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
	
	RabbitMQHost string
	RabbitMQPort int

	RabbitMQUser     string
	RabbitMQPassword string

	RabbitMQVHost    string
	RabbitMQExchange string

	RabbitMQAdminUser     string
	RabbitMQAdminPassword string
}

// LoadConfig parses key=value file
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer f.Close()

	cfg := &Config{}

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

	return cfg, scanner.Err()
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
	case "verbosity":
		c.Verbosity = parseInt(val)
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
