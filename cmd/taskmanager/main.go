package main

import (
	"flag"
	"fmt"
	"log"

	"taskmanager/internal/config"
	"taskmanager/internal/db"
	"taskmanager/internal/rabbitmq"
)

func atoi(s string) int {
	var v int
	fmt.Sscanf(s, "%d", &v)
	return v
}

type Runtime struct {
	Config *config.Config
}

func logAt(verbosity, level int, format string, args ...any) {
	if verbosity < level {
		return
	}
	log.Printf(format, args...)
}

func main() {

	// STEP 1: FIRST PASS CLI (ONLY config path)
	configPath := flag.String("c", "", "config file path")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("config file path is required (-c)")
	}

	// STEP 2: LOAD CONFIG FILE
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// RUNTIME OBJECT
	rt := &Runtime{Config: cfg}
	start(rt)
}

func start(rt *Runtime) {

	cfg := rt.Config

	//RABBITMQ Setup
	rabbitmq.NewAdmin(rabbitmq.Config{
		Host:     cfg.RabbitMQHost,
		Port:     cfg.RabbitMQPort,
		User:     cfg.RabbitMQUser,
		Password: cfg.RabbitMQPassword,
		Vhost:    cfg.RabbitMQVHost,
		Exchange: cfg.RabbitMQExchange,

		AdminUser:     cfg.RabbitMQAdminUser,
		AdminPassword: cfg.RabbitMQAdminPassword,
		Verbosity:     cfg.Verbosity,
	})

	if err := rabbitmq.Bootstrap(); err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	logAt(cfg.Verbosity, 1, "bootstrap complete")

	if err := rabbitmq.SetupTopology(); err != nil {
		log.Fatal(err)
	}

	logAt(cfg.Verbosity, 2, "starting rabbitmq consumers")

	if err := rabbitmq.StartConsumers(); err != nil {
		log.Fatalf("rabbitmq consumers failed: %v", err)
	}

	//DB Setup
	if err := db.Init(&db.Config{
		MySQLHost:     cfg.MySQLHost,
		MySQLPort:     cfg.MySQLPort,
		MySQLUser:     cfg.MySQLUser,
		MySQLPassword: cfg.MySQLPassword,
		MySQLDatabase: cfg.MySQLDatabase,
		MySQLSchema:   cfg.MySQLSchema,

		MySQLMaxOpenConns:    cfg.MySQLMaxOpenConns,
		MySQLMaxIdleConns:    cfg.MySQLMaxIdleConns,
		MySQLConnMaxLifetime: cfg.MySQLConnMaxLifetime,

		PartitionDays:        cfg.PartitionDays,
		RetentionPeriod:      cfg.RetentionPeriod,
		RetentionCleanupHour: cfg.RetentionCleanupHour,
		Verbosity:            cfg.Verbosity,

		// DB Workers
		DBWorkers:   cfg.DBWorkers,
		DBQueueSize: cfg.DBQueueSize,
	}); err != nil {
		log.Fatal(err)
	}

	for {
	}
}
