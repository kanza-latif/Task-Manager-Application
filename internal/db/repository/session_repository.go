package repository

import (
	"database/sql"
	"fmt"
	"taskmanager/internal/db/models"
	"time"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// List returns sessions with an optional limit.
func (r *SessionRepository) List(limit int) ([]*models.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, msisdn, site, private_ip, COALESCE(public_ip, ''),
		 COALESCE(ipv6, ''), COALESCE(start_port, 0), COALESCE(end_port, 0),
		 packets, wl_status, start_time, end_time
		 FROM session_table ORDER BY end_time DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("session list: %w", err)
	}
	defer rows.Close()
	return scanSessionRows(rows)
}

// FindByMSISDN returns all sessions for a given MSISDN.
func (r *SessionRepository) FindByMSISDN(msisdn string) ([]*models.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, msisdn, site, private_ip, COALESCE(public_ip, ''),
		 COALESCE(ipv6, ''), COALESCE(start_port, 0), COALESCE(end_port, 0),
		 packets, wl_status, start_time, end_time
		 FROM session_table WHERE msisdn = ? ORDER BY end_time DESC`, msisdn,
	)
	if err != nil {
		return nil, fmt.Errorf("session find by msisdn: %w", err)
	}
	defer rows.Close()
	return scanSessionRows(rows)
}

// FindByTimeRange returns sessions within a start and end time.
func (r *SessionRepository) FindByTimeRange(from, to time.Time) ([]*models.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, msisdn, site, private_ip, COALESCE(public_ip, ''),
		 COALESCE(ipv6, ''), COALESCE(start_port, 0), COALESCE(end_port, 0),
		 packets, wl_status, start_time, end_time
		 FROM session_table WHERE end_time BETWEEN ? AND ?
		 ORDER BY end_time DESC`, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("session find by time range: %w", err)
	}
	defer rows.Close()
	return scanSessionRows(rows)
}

// helpers

func scanSessionRows(rows *sql.Rows) ([]*models.Session, error) {
	var results []*models.Session
	for rows.Next() {
		s := &models.Session{}
		var privRaw, pubRaw []byte
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.MSISDN, &s.Site,
			&privRaw, &pubRaw,
			&s.IPv6, &s.StartPort, &s.EndPort,
			&s.Packets, &s.WLStatus,
			&s.StartTime, &s.EndTime,
		); err != nil {
			return nil, fmt.Errorf("session scan: %w", err)
		}
		s.PrivateIP = rawToIP(privRaw)
		s.PublicIP = rawToIP(pubRaw)
		results = append(results, s)
	}
	return results, rows.Err()
}
