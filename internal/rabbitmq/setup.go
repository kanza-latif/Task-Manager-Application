package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

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

		logAt(3, "declaring queue: %s", q)

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
			logError("error declaring queue %s: %v", q, err)
			return err
		}

		logAt(3, "binding queue: %s", q)

		err = GlobalClient.ch.QueueBind(
			q,
			q, // routing key = queue name
			GlobalClient.cfg.Exchange,
			false,
			nil,
		)

		if err != nil {
			logError("error binding queue %s: %v", q, err)
			return err
		}
	}

	logAt(1, "topology ready; queues=%d", len(Queues))
	return nil
}
