package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"taskmanager/internal/domain"
)

type AlarmRepository struct {
	db *sql.DB
}

func NewAlarmRepository(db *sql.DB) *AlarmRepository {
	return &AlarmRepository{db: db}
}

func (r *AlarmRepository) IsAlive() bool {
	// Use a short timeout so your application health check doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Ping returns an error if the database cannot be reached
	if err := r.db.PingContext(ctx); err != nil {
		return false
	}
	
	return true
}

// -------------------- CREATE --------------------

func (r *AlarmRepository) Create(a *domain.Alarm) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO alarm_table 
		(alarm_id, block_id, site, time_raised, severity, module, message, status, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.AlarmID,
		a.BlockID,
		a.Site,
		a.TimeRaised,
		a.Severity,
		a.Module,
		a.Message,
		a.Status,
		a.UpdatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("alarm create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("alarm create lastInsertId: %w", err)
	}

	return id, nil
}

// -------------------- READ (single) --------------------

func (r *AlarmRepository) GetByID(id int64) (*domain.Alarm, error) {
	row := r.db.QueryRow(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE id = ?`,
		id,
	)

	a := &domain.Alarm{}
	err := row.Scan(
		&a.ID,
		&a.AlarmID,
		&a.BlockID,
		&a.Site,
		&a.TimeRaised,
		&a.Severity,
		&a.Module,
		&a.Message,
		&a.Status,
		&a.UpdatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("alarm get by id: %w", err)
	}

	return a, nil
}

// -------------------- UPDATE --------------------

func (r *AlarmRepository) Update(a *domain.Alarm) error {
	_, err := r.db.Exec(
		`UPDATE alarm_table SET
			alarm_id = ?,
			block_id = ?,
			site = ?,
			time_raised = ?,
			severity = ?,
			module = ?,
			message = ?,
			status = ?,
			updated_by = ?
		WHERE id = ?`,
		a.AlarmID,
		a.BlockID,
		a.Site,
		a.TimeRaised,
		a.Severity,
		a.Module,
		a.Message,
		a.Status,
		a.UpdatedBy,
		a.ID,
	)

	if err != nil {
		return fmt.Errorf("alarm update: %w", err)
	}

	return nil
}

// -------------------- DELETE --------------------

func (r *AlarmRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM alarm_table WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("alarm delete: %w", err)
	}
	return nil
}

// -------------------- LIST (ALL) --------------------

func (r *AlarmRepository) List() ([]*domain.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table ORDER BY time_raised DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm list: %w", err)
	}
	defer rows.Close()

	return scanAlarmRows(rows)
}

func (r *AlarmRepository) FindByTimeRange(start, end time.Time) ([]*domain.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, Module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table 
		 WHERE time_raised BETWEEN ? AND ?
		 ORDER BY time_raised DESC`,
		start,
		end,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by time range: %w", err)
	}
	defer rows.Close()

	return scanAlarmRows(rows)
}

// -------------------- FILTERS (ENUM SAFE) --------------------

func (r *AlarmRepository) FindBySeverity(severity domain.AlarmSeverity) ([]*domain.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE severity = ? ORDER BY time_raised DESC`,
		severity,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by severity: %w", err)
	}
	defer rows.Close()

	return scanAlarmRows(rows)
}

func (r *AlarmRepository) FindByStatus(status domain.AlarmStatus) ([]*domain.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE status = ? ORDER BY time_raised DESC`,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by status: %w", err)
	}
	defer rows.Close()

	return scanAlarmRows(rows)
}

func (r *AlarmRepository) FindBySite(site string) ([]*domain.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE site = ? ORDER BY time_raised DESC`,
		site,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by site: %w", err)
	}
	defer rows.Close()

	return scanAlarmRows(rows)
}

func (r *AlarmRepository) FindByModule(module domain.AlarmModule) ([]*domain.Alarm, error) {
	rows, err := r.db.Query(
		`SELECT id, alarm_id, block_id, site, time_raised, severity, module,
		 COALESCE(message, ''), status, updated_by
		 FROM alarm_table WHERE module = ? ORDER BY time_raised DESC`,
		module,
	)
	if err != nil {
		return nil, fmt.Errorf("alarm find by module: %w", err)
	}
	defer rows.Close()

	return scanAlarmRows(rows)
}

// -------------------- SHARED SCANNER --------------------

func scanAlarmRows(rows *sql.Rows) ([]*domain.Alarm, error) {
	var results []*domain.Alarm
	for rows.Next() {
		a := &domain.Alarm{}
		if err := rows.Scan(
			&a.ID,
			&a.AlarmID,
			&a.BlockID,
			&a.Site,
			&a.TimeRaised,
			&a.Severity,
			&a.Module,
			&a.Message,
			&a.Status,
			&a.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("alarm scan: %w", err)
		}
		results = append(results, a)
	}
	return results, rows.Err()
}
