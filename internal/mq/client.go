package mq

import (
	"fmt"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Config holds RabbitMQ connection settings.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}

func LoadConfig() Config {
	return Config{
		Host:     getEnv("RABBITMQ_HOST", "127.0.0.1"),
		Port:     getEnv("RABBITMQ_PORT", "5672"),
		User:     getEnv("RABBITMQ_USER", "guest"),
		Password: getEnv("RABBITMQ_PASSWORD", "guest"),
		VHost:    getEnv("RABBITMQ_VHOST", "/"),
	}
}

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     Config
}

// Connect establishes a connection to RabbitMQ and declares the exchange.
func Connect(cfg Config) (*Client, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.VHost,
	)

	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ at %s:%s — %w", cfg.Host, cfg.Port, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the exchange — passive=false means create if not exists
	err = ch.ExchangeDeclare(
		Exchange, // name
		"topic",  // type — topic allows routing key pattern matching
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	fmt.Printf("Connected to RabbitMQ at %s:%s (vhost: %s)\n", cfg.Host, cfg.Port, cfg.VHost)

	return &Client{conn: conn, channel: ch, cfg: cfg}, nil
}

func (c *Client) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
