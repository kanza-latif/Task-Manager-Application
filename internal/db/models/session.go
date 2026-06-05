package models

import "time"

type Session struct {
	ID        int64
	SessionID string
	MSISDN    string
	Site      string
	PrivateIP string
	PublicIP  string // nullable
	IPv6      string // nullable
	StartPort int    // nullable
	EndPort   int    // nullable
	Packets   int
	WLStatus  bool
	StartTime time.Time
	EndTime   time.Time
}
