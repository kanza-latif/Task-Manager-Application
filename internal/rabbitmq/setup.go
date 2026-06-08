package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SetupTopology() error {
	err := GlobalClient.ch.ExchangeDeclare(
		GlobalClient.cfg.Exchange,
		"topic",
		true,  // durable
		false, // auto-delete
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for _, q := range Queues {

		log.Printf("declaring queue: %s", q)

		_, err := GlobalClient.ch.QueueDeclare(
			q,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false,
			amqp.Table{
				"x-queue-type": "quorum",
			},
		)

		if err != nil {
			log.Printf("error declaring queue: %s", q)
			return err
		}
		
		log.Printf("binding queue: %s", q)

		err = GlobalClient.ch.QueueBind(
			q,
			q, // routing key = queue name
			GlobalClient.cfg.Exchange,
			false,
			nil,
		)

		if err != nil {
			log.Printf("error binding queue: %s", q)
			return err
		}
	}

	return nil
}