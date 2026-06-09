package domain

import "time"

type AlarmSeverity string
type AlarmModule string
type AlarmStatus string

const (
	SeverityCritical AlarmSeverity = "critical"
	SeverityHigh     AlarmSeverity = "high"
	SeverityMedium   AlarmSeverity = "medium"
	SeverityLow      AlarmSeverity = "low"

	ModuleNetwork     AlarmModule = "network"
	ModuleSystem      AlarmModule = "system"
	ModuleApplication AlarmModule = "application"

	StatusPending    AlarmStatus = "pending"
	StatusInProgress AlarmStatus = "in_progress"
	StatusResolved   AlarmStatus = "resolved"
)

type Alarm struct {
	ID         int64
	AlarmID    string
	BlockID    string
	Site       string
	TimeRaised time.Time
	Severity   AlarmSeverity
	Module     AlarmModule
	Message    string
	Status     AlarmStatus
	UpdatedBy  int
}
