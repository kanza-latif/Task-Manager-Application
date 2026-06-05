package repository

import (
	"database/sql"
	"fmt"
	"taskmanager/internal/db/models"
)

type AlarmRepository struct {
	db *sql.DB
}

func NewAlarmRepository(db *sql.DB) *AlarmRepository {
	return &AlarmRepository{db: db}
}

// List returns all alarms ordered by time raised in descending order.
func (r *AlarmRepository) List() ([]*models.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, Module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table ORDER BY time_raised DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm list: %w", err)
	}
	defer rows.Close()
	return scanAlarmRows(rows)
}

// FindBySeverity returns alarms filtered by severity.
func (r *AlarmRepository) FindBySeverity(severity string) ([]*models.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, Module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE severity = ? ORDER BY time_raised DESC`, severity,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by severity: %w", err)
	}
	defer rows.Close()
	return scanAlarmRows(rows)
}

// FindByStatus returns alarms filtered by status.
func (r *AlarmRepository) FindByStatus(status string) ([]*models.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, Module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE status = ? ORDER BY time_raised DESC`, status,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by status: %w", err)
	}
	defer rows.Close()
	return scanAlarmRows(rows)
}

// FindBySite returns alarms for a specific site.
func (r *AlarmRepository) FindBySite(site string) ([]*models.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, Module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE site = ? ORDER BY time_raised DESC`, site,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by site: %w", err)
	}
	defer rows.Close()
	return scanAlarmRows(rows)
}

// helpers

func scanAlarmRows(rows *sql.Rows) ([]*models.Alarm, error) {
	var results []*models.Alarm
	for rows.Next() {
		a := &models.Alarm{}
		if err := rows.Scan(
			&a.ID, &a.AlarmID, &a.BlockID, &a.Site,
			&a.TimeRaised, &a.Severity, &a.Module,
			&a.Message, &a.Status, &a.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("alarm scan: %w", err)
		}
		results = append(results, a)
	}
	return results, rows.Err()
}
