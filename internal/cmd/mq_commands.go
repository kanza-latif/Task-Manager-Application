package cmd

import (
	"fmt"

	"taskmanager/internal/db"
	"taskmanager/internal/db/repository"
	"taskmanager/internal/mq"

	"github.com/spf13/cobra"
)

var mqCmd = &cobra.Command{
	Use:   "mq",
	Short: "RabbitMQ operations",
}

// mq publish cgnat — reads cgnat_table from MySQL and publishes to RabbitMQ
var mqPublishCGNATCmd = &cobra.Command{
	Use:     "publish-cgnat",
	Short:   "Publish full cgnat_table to RabbitMQ",
	Example: `  tasks mq publish-cgnat`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// connect to RabbitMQ
		mqCfg := mq.LoadConfig()
		mqClient, err := mq.Connect(mqCfg)
		if err != nil {
			return err
		}
		defer mqClient.Close()

		// fetch all cgnat entries from MySQL
		repo := repository.NewCGNATRepository(dbConn)
		entries, err := repo.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No CGNAT entries found in database.")
			return nil
		}

		// convert to MQ message format
		var msgs []mq.CGNATMessage
		for _, e := range entries {
			msgs = append(msgs, mq.CGNATMessage{
				ID:        e.ID,
				PrivateIP: e.PrivateIP,
				PublicIP:  e.PublicIP,
				StartPort: e.StartPort,
				EndPort:   e.EndPort,
			})
		}

		return mqClient.PublishCGNATLoad(msgs)
	},
}

// mq publish-whitelist — reads whitelist_table from MySQL and publishes to RabbitMQ
var mqPublishWhitelistCmd = &cobra.Command{
	Use:     "publish-whitelist",
	Short:   "Publish full whitelist_table to RabbitMQ",
	Example: `  tasks mq publish-whitelist`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mqCfg := mq.LoadConfig()
		mqClient, err := mq.Connect(mqCfg)
		if err != nil {
			return err
		}
		defer mqClient.Close()

		repo := repository.NewWhitelistRepository(dbConn)
		entries, err := repo.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No whitelist entries found in database.")
			return nil
		}

		var msgs []mq.WhitelistMessage
		for _, e := range entries {
			msgs = append(msgs, mq.WhitelistMessage{
				ID:     e.ID,
				MSISDN: e.MSISDN,
			})
		}

		return mqClient.PublishWhitelistLoad(msgs)
	},
}

// mq stats — consumes session.stats messages from RabbitMQ and displays them
var mqStatsCmd = &cobra.Command{
	Use:     "stats",
	Short:   "Consume and display session stats from RabbitMQ",
	Example: `  tasks mq stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mqCfg := mq.LoadConfig()
		mqClient, err := mq.Connect(mqCfg)
		if err != nil {
			return err
		}
		defer mqClient.Close()

		return mqClient.ConsumeStats()
	},
}

// mq watch — watches MySQL binary log and publishes changes to RabbitMQ
var mqWatchCmd = &cobra.Command{
	Use:     "watch",
	Short:   "Watch cgnat/whitelist tables for changes and publish to RabbitMQ",
	Example: `  tasks mq watch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mqCfg := mq.LoadConfig()
		mqClient, err := mq.Connect(mqCfg)
		if err != nil {
			return err
		}
		defer mqClient.Close()

		dbCfg := db.LoadConfig()
		listener := mq.NewBinlogListener(mqClient, dbCfg)
		return listener.Start(dbConn)
	},
}

// RegisterMQCommands wires up all mq sub-commands into the root.
func RegisterMQCommands() {
	mqCmd.AddCommand(
		mqPublishCGNATCmd,
		mqPublishWhitelistCmd,
		mqStatsCmd,
		mqWatchCmd,
	)
	rootCmd.AddCommand(mqCmd)
}
