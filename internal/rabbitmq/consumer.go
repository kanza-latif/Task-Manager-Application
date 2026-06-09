package rabbitmq

import (
	"encoding/json"
	"time"
)

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
	MultiSessionID   string
	CallingStationID string

	FramedIPv4 string
	PublicIPv4 string
	FramedIPv6 string

	PortStart uint16
	PortEnd   uint16

	SessionStart time.Time
	SessionEnd   time.Time

	ExtraAVPs []ExtraAVP
}

func StartConsumers() error {

	if err := startEntryConsumer(); err != nil {
		return err
	}

	return nil
}

func startEntryConsumer() error {

	msgs, err := GlobalClient.ch.Consume(
		RouteSessionFinal,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var s UserSession

			if err := json.Unmarshal(d.Body, &s); err != nil {
				logError("failed to decode %s message: %v", RouteSessionFinal, err)
				continue
			}

			logAt(1, "consumed final session stat")
			logAt(4, "session payload: %+v", s)
		}
	}()

	return nil
}
