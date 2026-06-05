package mq

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ConsumeStats listens on the session.stats queue and prints
// incoming stats messages to the terminal until Ctrl+C is pressed.
func (c *Client) ConsumeStats() error {
	// Declare a queue bound to session.stats routing key
	q, err := c.channel.QueueDeclare(
		"taskmanager.session.stats", // queue name
		true,                        // durable
		false,                       // auto-delete
		false,                       // exclusive
		false,                       // no-wait
		nil,                         // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare stats queue: %w", err)
	}

	// Bind queue to exchange with session.stats routing key
	err = c.channel.QueueBind(
		q.Name,            // queue name
		RouteSessionStats, // routing key
		Exchange,          // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind stats queue: %w", err)
	}

	msgs, err := c.channel.Consume(
		q.Name,          // queue
		"session.stats", // consumer tag
		true,            // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	fmt.Printf("Listening for stats on [%s]... Press Ctrl+C to stop\n\n", RouteSessionStats)
	printStatsHeader()

	// Graceful shutdown on Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("stats consumer channel closed")
			}
			handleStatsMessage(msg)

		case <-stop:
			fmt.Println("\nStopped consuming stats.")
			return nil
		}
	}
}

// handleStatsMessage parses and prints a single stats message.
func handleStatsMessage(msg amqp.Delivery) {
	var stats StatsMessage
	if err := json.Unmarshal(msg.Body, &stats); err != nil {
		fmt.Printf("Failed to parse stats message: %s\n", err)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\n",
		stats.Timestamp.Format(time.RFC822),
		stats.Site,
		stats.TotalPackets,
		stats.SessionCount,
		stats.WLCount,
	)
	w.Flush()
}

func printStatsHeader() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tSITE\tTOTAL PACKETS\tSESSIONS\tWHITELISTED")
	fmt.Fprintln(w, "---------\t----\t-------------\t--------\t-----------")
	w.Flush()
}
