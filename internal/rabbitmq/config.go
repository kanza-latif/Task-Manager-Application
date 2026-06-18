package rabbitmq

import (
	"log"
	"sync"

	"github.com/go-resty/resty/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	RouteSessionStart = "session.start"
	RouteSessionStop  = "session.stop"
	RouteSessionFinal = "session.final"

	RouteSessionStats  = "session.stats"
	RouteCGNATLoad     = "bootstrap.cgnat"
	RouteWhitelistLoad = "bootstrap.whitelist"

	RouteNodeHeartbeat = "node.heartbeat"
)

var Queues = []string{
	RouteSessionStart,
	RouteSessionStop,
	RouteSessionFinal,
	RouteSessionStats,
	RouteCGNATLoad,
	RouteWhitelistLoad,
	RouteNodeHeartbeat,
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Vhost    string
	Exchange string

	AdminUser     string
	AdminPassword string

	Verbosity int
	SiteName  string
	NodeName  string
}

type Admin struct {
	host   string
	client *resty.Client
}

type Client struct {
	cfg Config

	conn *amqp.Connection
	ch   *amqp.Channel

	admin Admin
}

var (
	GlobalClient *Client
	initOnce     sync.Once
)

func logAt(level int, format string, args ...any) {
	if GlobalClient == nil || GlobalClient.cfg.Verbosity < level {
		return
	}
	log.Printf("[rabbitmq] "+format, args...)
}

func logError(format string, args ...any) {
	log.Printf("[rabbitmq] ERROR: "+format, args...)
}
