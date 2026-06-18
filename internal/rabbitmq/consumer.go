package rabbitmq

import (
	"encoding/json"
	"taskmanager/internal/db"
	"taskmanager/internal/domain"
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

			db.GlobalConn.Session.Create(
				&domain.Session{
					SessionID: s.AccountSessionID,
					MSISDN:    s.CallingStationID,
					Site:      GlobalClient.cfg.SiteName,
					PrivateIP: s.FramedIPv4,
					PublicIP:  s.PublicIPv4,
					IPv6:      s.FramedIPv6,
					StartPort: int(s.PortStart),
					EndPort:   int(s.PortEnd),
					Packets:   int(s.PacketCount),
					WLStatus:  s.IsWhitelist,
					StartTime: s.SessionStart,
					EndTime:   s.SessionEnd,
				},
			)
		}
	}()

	return nil
}
