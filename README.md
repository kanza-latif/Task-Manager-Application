# MDF Task Manager

## Overview

MDF Task Manager is the central infrastructure and data management service of the MDF platform.

It is responsible for:

* Creating and maintaining the MDF MySQL database.
* Creating and maintaining RabbitMQ user, vhost, exchanges, queues, and bindings.
* Managing all database access through repositories.
* Loading bootstrap data into RabbitMQ which includes cgnat and whitelist data from database.
* Acting as the authoritative owner of MDF platform data.

Unlike other MDF applications, Task Manager is responsible for infrastructure ownership.

Other MDF services:

* Communicate with each other using RabbitMQ.
* Do not create or manage database structures.
* Do not own RabbitMQ resources.
* Access MDF database through Task Manager.

## Architecture

```text
                     ┌──────────────────────┐
                     │    Task Manager      │
                     │                      │
              ┌──────│ DB Owner             │
              |      │ RabbitMQ Owner       │
              |      └──────────┬───────────┘
              |                 │
              ▲          bootstrap.cgnat
              |          bootstrap.whitelist
       session.final            |
              |                 │
              |                 ▼
┌───────────────────────────────────────────────────┐
│                    Radius Parser                  │
│                                                   │
│ Capture → Parse → Session Engine → RabbitMQ       │
│                                                   │
└─────────────┬──────────────────────┬──────────────┘
              │                      │
              ▼                      ▲
      session.start                  |
      session.stop             node.heartbeat
              |                session.stats
              │                      │
              ▼                      │
     ┌─────────────────────┐         │
     │ Filtering Apps      │─────────┘
     │ (1..N instances)    │
     └─────────────────────┘
              │
              ▼

        Task Manager
```

## Responsibilities

### Database Management

Task Manager owns the MDF database lifecycle.

Responsibilities include:

* Database creation.
* Table creation.
* Schema initialization.
* Partition management.
* Partition cleanup.
* Connection management.
* Repository management.
* Data persistence.

#### Managed Tables

| Table           | Description          |
| --------------- | -------------------- |
| user_table      | User management      |
| session_table   | Session information  |
| alarm_table     | Alarm management     |
| cgnat_table     | CGNAT mappings       |
| whitelist_table | Subscriber whitelist |

### RabbitMQ Management

Task Manager owns all RabbitMQ resources required by MDF.

Responsibilities include:

* User creation.
* Vhost creation.
* Exchange creation.
* Queue creation.
* Queue bindings.
* Publisher initialization.
* Consumer initialization.
* RabbitMQ connection management.

#### Managed Queues

| Queue               | Description                           |
| ------------------- | ------------------------------------- |
| session.start       | Session start events                  |
| session.stop        | Session stop events                   |
| session.stats       | Session statistics                    |
| session.final       | Session completion events             |
| bootstrap.cgnat     | CGNAT bootstrap data                  |
| bootstrap.whitelist | Whitelist bootstrap data              |
| node.heartbeat      | Filtering node liveness monitoring    |

### Bootstrap Data Distribution

Task Manager distributes MDF reference data to the platform through RabbitMQ.

#### CGNAT Bootstrap

At startup, Task Manager:

1. Reads CGNAT mappings from MySQL.
2. Publishes mappings to:

```text
bootstrap.cgnat
```

#### Whitelist Bootstrap

At startup, Task Manager:

1. Reads whitelist entries from MySQL.
2. Publishes entries to:

```text
bootstrap.whitelist
```

This allows MDF applications to populate local caches without direct database access.

## Project Structure

```text
task-manager/
├── cmd
│   └── taskmanager
│       └── main.go
│
├── config
│   └── taskmanager.conf
│
├── internal
│   ├── config
│   │   └── config.go
│   │
│   ├── db
│   │   ├── db.go
│   │   ├── repository_wrapper.go
│   │   └── setup.go
│   │
│   ├── domain
│   │   ├── alarm.go
│   │   ├── cgnat.go
│   │   ├── session.go
│   │   ├── user.go
│   │   └── whitelist.go
│   │
│   ├── rabbitmq
│   │   ├── admin.go
│   │   ├── client.go
│   │   ├── config.go
│   │   ├── consumer.go
│   │   ├── producer.go
│   │   └── setup.go
│   │
│   └── repository
│       ├── alarm_repository.go
│       ├── cgnat_repository.go
│       ├── session_repository.go
│       ├── user_repository.go
│       └── whitelist_repository.go
│
├── scripts
│   ├── cluster_setup.sh
│   └── schema.sql
│
├── go.mod
├── go.sum
└── README.md
```

## Components

### cmd/taskmanager

Application entry point.

Responsible for:

* Loading configuration.
* Initializing RabbitMQ.
* Starting consumers and workers.
* Initializing database services.
* Initializing repositories.
* Sending bootstrap data to queues.

### internal/config

Configuration management.

Loads:

* MySQL configuration.
* RabbitMQ configuration.
* Runtime parameters.

### internal/db

Database infrastructure layer.

Responsibilities:

* MySQL connection management.
* Database creation.
* Schema setup.
* Partition creation.
* cleanup after retention period.
* Repository registration.
* Transaction management.

### internal/domain

Domain entities used throughout the platform.

Current entities:

* User
* Session
* Alarm
* CGNAT
* Whitelist

### internal/repository

Repository layer.

Contains all CRUD operations for MDF tables.

Repositories provide the only supported mechanism for interacting with MDF data.

### internal/rabbitmq

RabbitMQ infrastructure layer.

Responsibilities:

* User management.
* Vhost management.
* Connection management.
* Exchange creation.
* Queue creation.
* Queue binding.
* Message publishing.
* Message consumption.

## Data Flow

### Database Access

All MDF data is owned by Task Manager.

```text
MDF Application
        │
        ▼
 Task Manager
        │
        ▼
      MySQL
```

Applications never communicate directly with MySQL.

### Inter-Service Communication

Applications communicate through RabbitMQ.

```text
Application A
      │
      ▼
   RabbitMQ
      ▲
      │
Application B
```

Task Manager creates and manages the messaging infrastructure but is not required to be in every message path.

### Bootstrap Flow

```text
          MySQL
             │
             ▼
      Task Manager
             │
     ┌───────┴────────┐
     │                │
     ▼                ▼
bootstrap.cgnat   bootstrap.whitelist
     │                │
     ▼                ▼
 MDF Services   MDF Services
```

## RabbitMQ Topology

### Exchange

```text
radius_exchange
```

### Exchange Type

```text
topic
```

### Queues

| Queue               | Producer               | Consumer               |
|---------------------|------------------------|------------------------|
| session.start       | Radius Parser          | Filtering Applications |
| session.stop        | Radius Parser          | Filtering Applications |
| session.final       | Radius Parser          | Task Manager           |
| session.stats       | Filtering Applications | Radius Parser          |
| bootstrap.cgnat     | Task Manager           | Radius Parser          |
| bootstrap.whitelist | Task Manager           | Radius Parser          |
| node.heartbeat      | Filtering Applications | Radius Parser          |

### Queue Purposes

| Queue               | Purpose                               |
|---------------------|---------------------------------------|
| session.start       | New subscriber session                |
| session.stop        | Session termination notification      |
| session.final       | Finalized session export              |
| session.stats       | Packet counters and BYE notifications |
| bootstrap.cgnat     | CGNAT lookup synchronization          |
| bootstrap.whitelist | Whitelist synchronization             |
| node.heartbeat      | Filtering node liveness monitoring    |

## Message Structures

### session.start

Routing Key:

```text
session.start
```

Structure:

```go
type StartSessionMessage struct {
    AccountSessionID string
    FramedIPv4       string
    PublicIPv4       string
    FramedIPv6       string
    PortStart        uint16
    PortEnd          uint16
    IsWhitelist      bool
    FramedIPv6Len    int
}
```

Example:

```json
{
  "account_session_id": "abc123",
  "framed_ipv4": "10.250.41.153",
  "public_ipv4": "5.38.72.0",
  "framed_ipv6": "2001:db8::1",
  "port_start": 1,
  "port_end": 6666,
  "is_whitelist": true,
  "framed_ipv6_len": 64
}
```

### session.stop

Routing Key:

```text
session.stop
```

Structure:

```go
type StopSessionMessage struct {
    AccountSessionID string
}
```

Example:

```json
{
  "account_session_id": "abc123"
}
```

### session.final

Routing Key:

```text
session.final
```

Structure:

```go
type ExtraAVP struct {
    Type  uint8
    Len   uint8
    Value []byte
}

type UserSession struct {
    EventTimestamp uint32
    PacketCount    uint32
    DestroyTime    uint32

    AccountStatusType uint8
    IsWhitelist       bool

    AccountSessionID string
    CallingStationID string

    FramedIPv4    string
    PublicIPv4    string
    FramedIPv6    string
    FramedIPv6Len int

    PortStart uint16
    PortEnd   uint16

    SessionStart time.Time
    SessionEnd   time.Time

    byeAcks int

    ExtraAVPs []ExtraAVP
}
```

### session.stats

Routing Key:

```text
session.stats
```

Structure:

```go
type StatsMessage struct {
    SessionID   string `json:"session_id"`
    PacketCount uint32 `json:"packet_count"`
    ByeSeen     bool   `json:"bye_seen"`
}
```

Example:

```json
{
  "session_id": "abc123",
  "packet_count": 1500,
  "bye_seen": true
}
```

Used by Radius Parser to update packet counters and determine when a session can be finalized.

### bootstrap.cgnat

Routing Key:

```text
bootstrap.cgnat
```

Structure:

```go
type CgnatEntry struct {
    InsideIP  string
    NatIP     string
    StartPort uint16
    EndPort   uint16
    delete    bool
}
```

Example:

```json
{
  "inside_ip": "10.250.41.153",
  "nat_ip": "5.38.72.0",
  "start_port": 1,
  "end_port": 6666,
  "delete": false
}
```

Used to populate and update the in-memory CGNAT lookup cache.

### bootstrap.whitelist

Routing Key:

```text
bootstrap.whitelist
```

Structure:

```go
type WhitelistInfo struct {
    MSISDN string
    Status bool
    delete bool
}
```

Example:

```json
{
  "msisdn": "971501234567",
  "status": true,
  "delete": false
}
```

Used to populate and update the in-memory whitelist cache.

### node.heartbeat

Routing Key:

```text
node.heartbeat
```

Structure:

```go
type HeartbeatMessage struct {
    NodeID    string    `json:"node_id"`
    TimeStamp time.Time `json:"time_stamp"`
}
```

Example:

```json
{
  "node_id": "radius-parser-siteA",
  "time_stamp": "2026-06-11T12:30:00Z"
}
```

Used by MDF applications to advertise liveness and health status to Radius Parser.

## Startup Sequence

```text
1. Load Configuration

2. Connect to RabbitMQ

3. Create User

4. Create Vhost

5. Create Exchange

6. Create Queues

7. Create Bindings

8. Start Consumers

9. Create Database (if missing)

10. Create Tables (if missing)

11. Create Partitions (if missing)

12. Initialize Repositories

13. Connect to MySQL

14. Publish Bootstrap Data

15. Enter Runtime State
```

## Build

### Download Dependencies

```bash
go mod tidy
```

### Compile

```bash
go build -o taskmanager ./cmd/taskmanager
```

### Run

```bash
./taskmanager -c <path/to/taskmanager.conf>
```

## Sample Config File

Task manager config file is as follows:

```text
# RabbitMQ 
rabbitmq_host=127.0.0.1
rabbitmq_port=5672
rabbitmq_user=radius_user
rabbitmq_password=radius_pass
rabbitmq_vhost=radius
rabbitmq_exchange=radius_exchange

rabbitmq_admin_user=guest
rabbitmq_admin_password=guest

# MySQL
mysql_host=192.168.20.139
mysql_port=3306
mysql_user=monitor
mysql_password=xflow@123
mysql_database=MDF
mysql_schema=./scripts/schema.sql

mysql_max_open_conns=50
mysql_max_idle_conns=10
mysql_conn_max_lifetime=3600

# DB Workers
db_workers=8
db_queue_size=100000

# Partition params
partition_days=3
retention_period=45v
# UTC hour, 0-23
retention_cleanup_hour=2

# Application - internal
verbosity=3
```
