package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func New() error {

	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		GlobalClient.cfg.User,
		GlobalClient.cfg.Password,
		GlobalClient.cfg.Host,
		GlobalClient.cfg.Port,
		GlobalClient.cfg.Vhost,
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err

	}

	if err != nil {
		ch.Close()
		conn.Close()
		return err

	}

	GlobalClient.conn = conn
	GlobalClient.ch = ch

	return err
}

func Close() {

	if GlobalClient == nil {
		return
	}

	if GlobalClient.ch != nil {
		_ = GlobalClient.ch.Close()
	}

	if GlobalClient.conn != nil {
		_ = GlobalClient.conn.Close()
	}

	GlobalClient = nil
}
