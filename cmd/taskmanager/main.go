package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	// "taskmanager/internal/cmd"
	"taskmanager/internal/config"
	"taskmanager/internal/rabbitmq"
)

func atoi(s string) int {
	var v int
	fmt.Sscanf(s, "%d", &v)
	return v
}

func overrideCLI(cfg *config.Config) {

	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {

		case "-v", "--verbose":
			if i+1 < len(args) {
				cfg.Verbosity = atoi(args[i+1])
				i++
			}

		case "--rabbitmq-host":
			if i+1 < len(args) {
				cfg.RabbitMQHost = args[i+1]
				i++
			}

		case "--rabbitmq-port":
			if i+1 < len(args) {
				cfg.RabbitMQPort = atoi(args[i+1])
				i++
			}

		case "--rabbitmq-user":
			if i+1 < len(args) {
				cfg.RabbitMQUser = args[i+1]
				i++
			}

		case "--rabbitmq-password":
			if i+1 < len(args) {
				cfg.RabbitMQPassword = args[i+1]
				i++
			}

		case "--rabbitmq-vhost":
			if i+1 < len(args) {
				cfg.RabbitMQVHost = args[i+1]
				i++
			}

		case "--rabbitmq-exchange":
			if i+1 < len(args) {
				cfg.RabbitMQExchange = args[i+1]
				i++
			}
		}
	}
}

type Runtime struct {
	Config *config.Config
}

func main() {

	// =========================
	// STEP 1: FIRST PASS CLI (ONLY config path)
	// =========================
	configPath := flag.String("c", "", "config file path")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("config file path is required (-c)")
	}

	// =========================
	// STEP 2: LOAD CONFIG FILE
	// =========================
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// =========================
	// STEP 3: SECOND PASS CLI OVERRIDES
	// (rebuild flag set manually for full control like C)
	// =========================
	overrideCLI(cfg)

	// =========================
	// RUNTIME OBJECT
	// =========================
	rt := &Runtime{Config: cfg}
	start(rt)
}

func start(rt *Runtime) {

	cfg := rt.Config
	// cmd.Execute()
	rabbitmq.NewAdmin(rabbitmq.Config{
		Host:     cfg.RabbitMQHost,
		Port:     cfg.RabbitMQPort,
		User:     cfg.RabbitMQUser,
		Password: cfg.RabbitMQPassword,
		Vhost:    cfg.RabbitMQVHost,
		Exchange: cfg.RabbitMQExchange,

		AdminUser:     cfg.RabbitMQAdminUser,
		AdminPassword: cfg.RabbitMQAdminPassword,
	})

	if err := rabbitmq.Bootstrap(); err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Printf("bootstrap complete")

	if err := rabbitmq.SetupTopology(); err != nil {
		log.Fatal(err)
	}

	log.Printf("starting topology bootstrap")

	if err := rabbitmq.StartConsumers(); err != nil {
		log.Fatalf("rabbitmq consumers failed: %v", err)
	}

	for {
    }
}
