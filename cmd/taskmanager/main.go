package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

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

var (
	rt       Runtime
	gRunning atomic.Bool
)

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
	rt.Config = cfg
	start()
}

func waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(
		sigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGSEGV,
	)
	sig := <-sigCh
	log.Printf("Received signal: %v", sig)
	shutdown()
	os.Exit(0)
}

func shutdown() {
	logAt(rt.Config.Verbosity, 1, "starting rabbitmq consumers")
	log.Println("Stopping Task Manager")
	gRunning.Store(false)
}

func start() {

	gRunning.Store(true)

	//RABBITMQ Setup
	rabbitmq.NewAdmin(rabbitmq.Config{
		Host:     rt.Config.RabbitMQHost,
		Port:     rt.Config.RabbitMQPort,
		User:     rt.Config.RabbitMQUser,
		Password: rt.Config.RabbitMQPassword,
		Vhost:    rt.Config.RabbitMQVHost,
		Exchange: rt.Config.RabbitMQExchange,

		AdminUser:     rt.Config.RabbitMQAdminUser,
		AdminPassword: rt.Config.RabbitMQAdminPassword,

		Verbosity: rt.Config.Verbosity,
		SiteName:  rt.Config.SiteName,
		NodeName:  rt.Config.NodeName,
	})

	if err := rabbitmq.Bootstrap(); err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	logAt(rt.Config.Verbosity, 1, "bootstrap complete")

	if err := rabbitmq.SetupTopology(); err != nil {
		log.Fatal(err)
	}

	logAt(rt.Config.Verbosity, 2, "starting rabbitmq consumers")

	if err := rabbitmq.StartConsumers(); err != nil {
		log.Fatalf("rabbitmq consumers failed: %v", err)
	}

	//DB Setup
	if err := db.Init(&db.Config{
		MySQLHost:     rt.Config.MySQLHost,
		MySQLPort:     rt.Config.MySQLPort,
		MySQLUser:     rt.Config.MySQLUser,
		MySQLPassword: rt.Config.MySQLPassword,
		MySQLDatabase: rt.Config.MySQLDatabase,
		MySQLSchema:   rt.Config.MySQLSchema,

		MySQLMaxOpenConns:    rt.Config.MySQLMaxOpenConns,
		MySQLMaxIdleConns:    rt.Config.MySQLMaxIdleConns,
		MySQLConnMaxLifetime: rt.Config.MySQLConnMaxLifetime,

		PartitionDays:        rt.Config.PartitionDays,
		RetentionPeriod:      rt.Config.RetentionPeriod,
		RetentionCleanupHour: rt.Config.RetentionCleanupHour,
		Verbosity:            rt.Config.Verbosity,

		// DB Workers
		DBWorkers:   rt.Config.DBWorkers,
		DBQueueSize: rt.Config.DBQueueSize,

		SiteName:  rt.Config.SiteName,
		NodeName:  rt.Config.NodeName,
	}); err != nil {
		log.Fatal(err)
	}

	if cgnat, whitelist, err := db.InitBootstrap(); err != nil {
		log.Fatal(err)
	} else {
		rabbitmq.PublishCGNAT(cgnat)
		rabbitmq.PublishWhitelist(whitelist)
	}

	for gRunning.Load() {
	}
}
