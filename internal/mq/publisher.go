package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishCGNATLoad publishes the full cgnat table as individual messages
// to the bootstrap.cgnat routing key.
func (c *Client) PublishCGNATLoad(entries []CGNATMessage) error {
	fmt.Printf("Publishing %d CGNAT entries to [%s]...\n", len(entries), RouteCGNATLoad)

	for _, entry := range entries {
		body, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal CGNAT entry %d: %w", entry.ID, err)
		}

		if err := c.publish(RouteCGNATLoad, body); err != nil {
			return fmt.Errorf("failed to publish CGNAT entry %d: %w", entry.ID, err)
		}
	}

	fmt.Printf("Published %d CGNAT entries to [%s]\n", len(entries), RouteCGNATLoad)
	return nil
}

// PublishWhitelistLoad publishes the full whitelist table as individual messages
// to the bootstrap.whitelist routing key.
func (c *Client) PublishWhitelistLoad(entries []WhitelistMessage) error {
	fmt.Printf("Publishing %d whitelist entries to [%s]...\n", len(entries), RouteWhitelistLoad)

	for _, entry := range entries {
		body, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal whitelist entry %d: %w", entry.ID, err)
		}

		if err := c.publish(RouteWhitelistLoad, body); err != nil {
			return fmt.Errorf("failed to publish whitelist entry %d: %w", entry.ID, err)
		}
	}

	fmt.Printf("Published %d whitelist entries to [%s]\n", len(entries), RouteWhitelistLoad)
	return nil
}

// PublishChangeEvent publishes a single change event (insert/update/delete)
// triggered by MySQL binary log changes on cgnat or whitelist tables.
func (c *Client) PublishChangeEvent(table, action string, data interface{}) error {
	event := ChangeEvent{
		Table:     table,
		Action:    action,
		Timestamp: time.Now(),
		Data:      data,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal change event: %w", err)
	}

	// Route to the correct queue based on table
	route := RouteCGNATLoad
	if table == "whitelist_table" {
		route = RouteWhitelistLoad
	}

	if err := c.publish(route, body); err != nil {
		return fmt.Errorf("failed to publish change event: %w", err)
	}

	fmt.Printf("Change event published — table: %s action: %s\n", table, action)
	return nil
}

// publish is the internal low-level publish method.
func (c *Client) publish(routingKey string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.channel.PublishWithContext(
		ctx,
		Exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survive broker restart
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}
