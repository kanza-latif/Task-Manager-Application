# MDF Task Manager — Developer Handover Guide

## Overview

This is a Go-based command-line (CLI) application that integrates with:
- **MySQL database** on a remote VM — stores and queries MDF operational data
- **RabbitMQ** on a remote server — publishes CGNAT/whitelist data and consumes session stats

The application is built for MDF (Mobile Data Filtering) Version 3.0 and provides a terminal interface to interact with all five MDF database tables and the RabbitMQ message broker.

---

## Table of Contents

1. [Project Structure](#1-project-structure)
2. [File Descriptions](#2-file-descriptions)
3. [Prerequisites](#3-prerequisites)
4. [MySQL Setup](#4-mysql-setup)
5. [RabbitMQ Setup](#5-rabbitmq-setup)
6. [Environment Variables](#6-environment-variables)
7. [Build Instructions](#7-build-instructions)
8. [All Commands Reference](#8-all-commands-reference)
9. [How Data Flows](#9-how-data-flows)
10. [Troubleshooting](#10-troubleshooting)

---

## 1. Project Structure

```
taskmanager/
│
├── main.go                          ← Entry point — starts the CLI
├── go.mod                           ← Go module file — lists all dependencies
│
├── migrations/
│   └── 001_create_tasks.sql         ← SQL to create the tasks table (run once on MySQL VM)
│
├── cmd/                             ← All CLI commands live here
│   ├── commands.go                  ← Root command + MySQL connection setup
│   ├── mdf_commands.go              ← Commands for all MDF database tables
│   └── mq_commands.go               ← Commands for RabbitMQ operations
│
└── internal/                        ← All business logic lives here
    ├── db/
    │   ├── db.go                    ← MySQL connection + database existence check
    │   ├── models/                  ← Data structs matching database table columns
    │   │   ├── task.go              ← Task struct
    │   │   ├── cgnat.go             ← CGNAT struct
    │   │   ├── whitelist.go         ← Whitelist struct
    │   │   ├── user.go              ← User struct
    │   │   ├── session.go           ← Session struct
    │   │   └── alarm.go             ← Alarm struct
    │   └── repository/              ← All MySQL queries live here (one file per table)
    │       ├── task_repository.go
    │       ├── cgnat_repository.go
    │       ├── whitelist_repository.go
    │       ├── user_repository.go
    │       ├── session_repository.go
    │       └── alarm_repository.go
    │
    └── mq/                          ← All RabbitMQ logic lives here
        ├── routes.go                ← Queue names and routing key constants
        ├── messages.go              ← JSON message structs for RabbitMQ payloads
        ├── client.go                ← RabbitMQ connection handler
        ├── publisher.go             ← Publishes CGNAT and whitelist data to RabbitMQ
        ├── consumer.go              ← Consumes session.stats messages from RabbitMQ
        └── binlog.go                ← Watches MySQL binary log and publishes changes
```

---

## 2. File Descriptions

### `main.go`
The entry point of the application. Contains only 3 lines — imports the `cmd` package and calls `cmd.Execute()`. Do not add logic here.

### `go.mod`
Defines the module name (`taskmanager`) and all external dependencies:
- `github.com/go-sql-driver/mysql` — MySQL driver
- `github.com/spf13/cobra` — CLI framework
- `github.com/rabbitmq/amqp091-go` — RabbitMQ client
- `github.com/go-mysql-org/go-mysql` — MySQL binary log listener

### `cmd/commands.go`
The foundation of the CLI. Defines the root `tasks` command. The `PersistentPreRunE` function runs before every command and connects to MySQL automatically. `Execute()` wires all sub-commands together and starts the app.

### `cmd/mdf_commands.go`
All commands that read from the MDF MySQL database tables: cgnat, whitelist, user, session, and alarm.

### `cmd/mq_commands.go`
All commands that interact with RabbitMQ: publish cgnat/whitelist data, consume stats, and watch the binary log.

### `internal/db/db.go`
Handles MySQL connection. Reads credentials from environment variables. On every startup it checks whether the target database exists — if not, it creates it automatically.

### `internal/db/models/`
Plain Go structs that mirror each database table column for column. No logic here — just data shapes.

### `internal/db/repository/`
All SQL queries. Each file is responsible for one table. The `cmd` files call these, and these talk to MySQL.

### `internal/mq/routes.go`
Defines all RabbitMQ routing keys as constants so they are never hardcoded anywhere else:
- `session.start` / `session.stop` / `session.final` — session lifecycle (MDF writes these)
- `session.stats` — task manager consumes this
- `bootstrap.cgnat` — task manager publishes this
- `bootstrap.whitelist` — task manager publishes this

### `internal/mq/client.go`
Establishes the RabbitMQ connection and declares the `radius_exchange` exchange. All other MQ files use this client.

### `internal/mq/publisher.go`
Reads data from MySQL (via repository) and publishes it as JSON messages to RabbitMQ queues.

### `internal/mq/consumer.go`
Listens to the `session.stats` queue and prints incoming stats to the terminal in real time.

### `internal/mq/binlog.go`
Uses MySQL binary logging (replication protocol) to detect any INSERT, UPDATE, or DELETE on `cgnat_table` or `whitelist_table` and immediately publishes a change event to RabbitMQ. This runs as a long-lived background watcher.

---

## 3. Prerequisites

On the machine running this application:
- Go 1.22 or higher — `go version`
- Network access to the MySQL VM at `192.168.20.139:3306`
- Network access to the RabbitMQ server at `127.0.0.1:5672`

---

## 4. MySQL Setup

### 4.1 — Verify All Tables Exist

Connect to MySQL and confirm all five tables are present:

```bash
mysql -u root1 -proot1_pass -h 192.168.20.139 MDF -e "SHOW TABLES;"
```

Expected output:
```
+-----------------+
| Tables_in_MDF   |
+-----------------+
| alarm_table     |
| cgnat_table     |
| session_table   |
| user_table      |
| whitelist_table |
+-----------------+
```

If any table is missing, run the schema file:
```bash
mysql -u root1 -proot1_pass -h 192.168.20.139 MDF < migrations/001_create_tasks.sql
```

### 4.2 — Enable Binary Logging (required for `mq watch`)

SSH into the MySQL VM and edit:
```bash
sudo nano /etc/mysql/mysql.conf.d/mysqld.cnf
```

Add under `[mysqld]`:
```ini
[mysqld]
server-id        = 1
log_bin          = /var/log/mysql/mysql-bin.log
binlog_format    = ROW
binlog_row_image = FULL
```

Restart MySQL:
```bash
sudo systemctl restart mysql
```

Verify:
```bash
mysql -u root1 -proot1_pass -h 192.168.20.139 MDF -e "SHOW MASTER STATUS;"
```
You should see a binlog filename and position. Empty result means binary logging is still off.

### 4.3 — Grant Replication Permission (required for `mq watch`)

```sql
mysql -u root1 -proot1_pass -h 192.168.20.139

GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'root1'@'%';
FLUSH PRIVILEGES;
```

---

## 5. RabbitMQ Setup

The RabbitMQ cluster is set up automatically by the included script. To verify queues are running:

```bash
# List all queues in the radius vhost
rabbitmqctl list_queues -p radius name messages consumers

# List exchanges
rabbitmqctl list_exchanges -p radius
```

Or open the management UI in a browser:
```
http://127.0.0.1:15672
Username: radius_user
Password: radius_pass
```

If the management UI is not available:
```bash
rabbitmq-plugins enable rabbitmq_management
sudo systemctl restart rabbitmq-server
```

---

## 6. Environment Variables

All credentials are read from environment variables. Never hardcoded in the source code.

Set them in the terminal before running the app:

```bash
# MySQL
export DB_HOST=192.168.20.139
export DB_PORT=3306
export DB_USER=root1
export DB_PASSWORD=root1_pass
export DB_NAME=MDF

# RabbitMQ
export RABBITMQ_HOST=127.0.0.1
export RABBITMQ_PORT=5672
export RABBITMQ_USER=radius_user
export RABBITMQ_PASSWORD=radius_pass
export RABBITMQ_VHOST=radius
```

To make permanent (survive terminal restarts), add to `~/.bashrc`:

```bash
echo 'export DB_HOST=192.168.20.139' >> ~/.bashrc
echo 'export DB_PORT=3306' >> ~/.bashrc
echo 'export DB_USER=root1' >> ~/.bashrc
echo 'export DB_PASSWORD=root1_pass' >> ~/.bashrc
echo 'export DB_NAME=MDF' >> ~/.bashrc
echo 'export RABBITMQ_HOST=127.0.0.1' >> ~/.bashrc
echo 'export RABBITMQ_PORT=5672' >> ~/.bashrc
echo 'export RABBITMQ_USER=radius_user' >> ~/.bashrc
echo 'export RABBITMQ_PASSWORD=radius_pass' >> ~/.bashrc
echo 'export RABBITMQ_VHOST=radius' >> ~/.bashrc
source ~/.bashrc
```

### Fallback defaults (if env vars are not set):

| Variable | Default |
|---|---|
| DB_HOST | 127.0.0.1 |
| DB_PORT | 3306 |
| DB_USER | root |
| DB_PASSWORD | (empty) |
| DB_NAME | taskmanager |
| RABBITMQ_HOST | 127.0.0.1 |
| RABBITMQ_PORT | 5672 |
| RABBITMQ_USER | guest |
| RABBITMQ_PASSWORD | guest |
| RABBITMQ_VHOST | / |

---

## 7. Build Instructions

Run these commands inside the `taskmanager/` folder:

```bash
# Step 1 — download all dependencies
go mod tidy

# Step 2 — compile into a binary called 'tasks'
go build -o tasks .

# Step 3 — verify it built correctly
./tasks --help
```

If `go mod tidy` fails due to network restrictions, try:
```bash
go env -w GOPROXY=direct
go mod tidy
```

Every time you change any `.go` file, you must rebuild:
```bash
go build -o tasks .
```

---

## 8. All Commands Reference

### Database Commands

#### CGNAT Table
```bash
./tasks cgnat list                        # List all CGNAT mappings
./tasks cgnat find-private 10.0.0.5       # Find mapping by private IP
./tasks cgnat find-public 203.0.113.10    # Find all mappings sharing a public IP
```

#### Whitelist Table
```bash
./tasks whitelist list                    # List all whitelisted MSISDNs
./tasks whitelist check 923001234567      # Check if an MSISDN is whitelisted
```

#### User Table
```bash
./tasks user list                         # List all users
./tasks user get admin1                   # Get details of a specific user
./tasks user filter --type admin          # Filter users by type (admin/viewer/whitelist)
```

#### Session Table
```bash
./tasks session list                      # List recent 50 sessions
./tasks session list --limit 100          # List recent N sessions
./tasks session find 923001234567         # Find all sessions for an MSISDN
./tasks session range --from "2026-05-19 00:00:00" --to "2026-05-19 23:59:59"
```

#### Alarm Table
```bash
./tasks alarm list                        # List all alarms
./tasks alarm severity --level critical   # Filter by severity (critical/high/medium/low)
./tasks alarm status --state pending      # Filter by status (pending/in_progress/resolved)
./tasks alarm site --name siteR           # Filter alarms by site name
```

### RabbitMQ Commands

```bash
# Read cgnat_table from MySQL and publish every row to bootstrap.cgnat queue
./tasks mq publish-cgnat

# Read whitelist_table from MySQL and publish every row to bootstrap.whitelist queue
./tasks mq publish-whitelist

# Connect to RabbitMQ and display incoming session.stats messages in real time
# Runs until Ctrl+C is pressed
./tasks mq stats

# Watch MySQL binary log for any INSERT/UPDATE/DELETE on cgnat_table or whitelist_table
# and automatically publish the change to the correct RabbitMQ queue
# Runs until Ctrl+C is pressed
./tasks mq watch
```

---

## 9. How Data Flows

```
                        ┌─────────────────────┐
                        │   MySQL VM           │
                        │   192.168.20.139     │
                        │                      │
                        │   MDF Database       │
                        │   ├── cgnat_table    │
                        │   ├── whitelist_table│
                        │   ├── session_table  │
                        │   ├── user_table     │
                        │   └── alarm_table    │
                        └────────┬─────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │ READ             │ BINLOG WATCH      │
              ▼                  ▼                   │
    ┌─────────────────┐   ┌────────────┐            │
    │  tasks cgnat    │   │ tasks mq   │            │
    │  tasks whitelist│   │ watch      │            │
    │  tasks session  │   └─────┬──────┘            │
    │  tasks user     │         │ publishes changes  │
    │  tasks alarm    │         │                   │
    └─────────────────┘         ▼                   │
                        ┌───────────────────┐       │
                        │   RabbitMQ        │       │
                        │   radius_exchange │       │
                        │                  │       │
                        │ bootstrap.cgnat  │◄──────┘
                        │ bootstrap.whitelist       │
                        │ session.stats    │        │
                        └───────┬───────────┘
                                │
                    ┌───────────┤
                    │ CONSUME   │ PUBLISH
                    ▼           ▼
            ┌─────────────┐  ┌──────────────────┐
            │ tasks mq    │  │ tasks mq          │
            │ stats       │  │ publish-cgnat     │
            │             │  │ publish-whitelist │
            └─────────────┘  └──────────────────┘
```

**Publish flow:**
`./tasks mq publish-cgnat` → reads `cgnat_table` from MySQL → publishes each row as JSON → `bootstrap.cgnat` queue on RabbitMQ

**Consume flow:**
`./tasks mq stats` → connects to RabbitMQ → listens on `session.stats` → prints live stats to terminal

**Binary log flow:**
`./tasks mq watch` → connects to MySQL replication stream → detects any change on `cgnat_table` or `whitelist_table` → publishes change event to the correct RabbitMQ queue automatically

---

## 10. Troubleshooting

### `DB connection failed`
- Check env vars are set: `echo $DB_HOST`
- Check MySQL is reachable: `ping 192.168.20.139`
- Check MySQL port: `telnet 192.168.20.139 3306`

### `Table does not exist`
- Run the schema SQL: `mysql -u root1 -proot1_pass -h 192.168.20.139 MDF < migrations/001_create_tasks.sql`

### `Failed to connect to RabbitMQ`
- Check env vars: `echo $RABBITMQ_HOST`
- Check RabbitMQ is running: `rabbitmqctl status`
- Check port: `telnet 127.0.0.1 5672`

### `SHOW MASTER STATUS returns empty`
- Binary logging is not enabled — follow Section 4.2

### `mq watch fails with replication error`
- Run the GRANT command in Section 4.3
- Ensure `binlog_format = ROW` in mysqld.cnf

### `go mod tidy fails`
- Try: `go env -w GOPROXY=direct && go mod tidy`
- Or: `go env -w GOPROXY=https://goproxy.cn,direct && go mod tidy`

### Changes not reflected after editing code
- You must rebuild: `go build -o tasks .`
