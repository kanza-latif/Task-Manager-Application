package mq

import "time"

// CGNATMessage is the payload published to bootstrap.cgnat
type CGNATMessage struct {
	ID        int64  `json:"id"`
	PrivateIP string `json:"private_ip"`
	PublicIP  string `json:"public_ip"`
	StartPort int    `json:"start_port"`
	EndPort   int    `json:"end_port"`
}

// WhitelistMessage is the payload published to bootstrap.whitelist
type WhitelistMessage struct {
	ID     int64  `json:"id"`
	MSISDN string `json:"msisdn"`
}

// StatsMessage is the payload consumed from session.stats
type StatsMessage struct {
	Site         string    `json:"site"`
	MSISDN       string    `json:"msisdn"`
	TotalPackets int       `json:"total_packets"`
	SessionCount int       `json:"session_count"`
	WLCount      int       `json:"wl_count"`
	Timestamp    time.Time `json:"timestamp"`
}

// ChangeEvent is published when cgnat or whitelist table changes via binary log
type ChangeEvent struct {
	Table     string      `json:"table"`  // "cgnat_table" or "whitelist_table"
	Action    string      `json:"action"` // "insert", "update", "delete"
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"` // CGNATMessage or WhitelistMessage
}
