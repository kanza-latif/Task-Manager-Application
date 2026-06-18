package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"taskmanager/internal/domain"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) IsAlive() bool {
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

func (r *SessionRepository) Create(s *domain.Session) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO session_table
		(session_id, msisdn, site, private_ip, public_ip, ipv6,
		 start_port, end_port, packets, wl_status, start_time, end_time)
		VALUES (?, ?, ?, INET6_ATON(?), INET6_ATON(?), ?, ?, ?, ?, ?, ?, ?)`,
		s.SessionID,
		s.MSISDN,
		s.Site,
		s.PrivateIP,
		nullString(s.PublicIP),
		nullString(s.IPv6),
		nullInt(s.StartPort),
		nullInt(s.EndPort),
		s.Packets,
		s.WLStatus,
		s.StartTime,
		s.EndTime,
	)
	if err != nil {
		return 0, fmt.Errorf("session create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("session create lastInsertId: %w", err)
	}

	return id, nil
}

// -------------------- BULK INSERT (CRITICAL PATH) --------------------

func (r *SessionRepository) BulkCreate(sessions []*domain.Session) error {
	if len(sessions) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("session bulk begin: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO session_table
		(session_id, msisdn, site, private_ip, public_ip, ipv6,
		 start_port, end_port, packets, wl_status, start_time, end_time)
		VALUES (?, ?, ?, INET6_ATON(?), INET6_ATON(?), ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("session bulk prepare: %w", err)
	}
	defer stmt.Close()

	for _, s := range sessions {
		_, err := stmt.Exec(
			s.SessionID,
			s.MSISDN,
			s.Site,
			s.PrivateIP,
			nullString(s.PublicIP),
			nullString(s.IPv6),
			nullInt(s.StartPort),
			nullInt(s.EndPort),
			s.Packets,
			s.WLStatus,
			s.StartTime,
			s.EndTime,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("session bulk exec: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("session bulk commit: %w", err)
	}

	return nil
}

// -------------------- READ (single) --------------------

func (r *SessionRepository) GetByID(id int64) (*domain.Session, error) {
	row := r.db.QueryRow(
		`SELECT id, session_id, msisdn, site, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, ipv6,
		        start_port, end_port, packets, wl_status, start_time, end_time
		 FROM session_table
		 WHERE id = ?`,
		id,
	)

	var s domain.Session
	err := row.Scan(
		&s.ID,
		&s.SessionID,
		&s.MSISDN,
		&s.Site,
		&s.PrivateIP,
		&s.PublicIP,
		&s.IPv6,
		&s.StartPort,
		&s.EndPort,
		&s.Packets,
		&s.WLStatus,
		&s.StartTime,
		&s.EndTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("session get by id: %w", err)
	}

	return &s, nil
}

// -------------------- LIST --------------------

func (r *SessionRepository) List(limit int) ([]*domain.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, msisdn, site, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, ipv6,
		        start_port, end_port, packets, wl_status, start_time, end_time
		 FROM session_table
		 ORDER BY end_time DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("session list: %w", err)
	}
	defer rows.Close()

	return scanSessionRows(rows)
}

// -------------------- TIME RANGE (MOST IMPORTANT QUERY) --------------------

func (r *SessionRepository) FindByTimeRange(start, end time.Time) ([]*domain.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, msisdn, site, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, ipv6,
		        start_port, end_port, packets, wl_status, start_time, end_time
		 FROM session_table
		 WHERE end_time BETWEEN ? AND ?
		 ORDER BY end_time DESC`,
		start,
		end,
	)
	if err != nil {
		return nil, fmt.Errorf("session time range: %w", err)
	}
	defer rows.Close()

	return scanSessionRows(rows)
}

// -------------------- FILTERS --------------------

func (r *SessionRepository) FindByMSISDN(msisdn string) ([]*domain.Session, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, msisdn, site, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, ipv6,
		        start_port, end_port, packets, wl_status, start_time, end_time
		 FROM session_table
		 WHERE msisdn = ?
		 ORDER BY end_time DESC`,
		msisdn,
	)
	if err != nil {
		return nil, fmt.Errorf("session by msisdn: %w", err)
	}
	defer rows.Close()

	return scanSessionRows(rows)
}

func (r *SessionRepository) FindBySessionID(sessionID string) (*domain.Session, error) {
	row := r.db.QueryRow(
		`SELECT id, session_id, msisdn, site, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, ipv6,
		        start_port, end_port, packets, wl_status, start_time, end_time
		 FROM session_table
		 WHERE session_id = ?`,
		sessionID,
	)

	var s domain.Session
	err := row.Scan(
		&s.ID,
		&s.SessionID,
		&s.MSISDN,
		&s.Site,
		&s.PrivateIP,
		&s.PublicIP,
		&s.IPv6,
		&s.StartPort,
		&s.EndPort,
		&s.Packets,
		&s.WLStatus,
		&s.StartTime,
		&s.EndTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("session by session_id: %w", err)
	}

	return &s, nil
}

// -------------------- DELETE --------------------

func (r *SessionRepository) Delete(id int64) error {
	_, err := r.db.Exec(
		`DELETE FROM session_table WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	return nil
}

// -------------------- HELPERS --------------------

func scanSessionRows(rows *sql.Rows) ([]*domain.Session, error) {
	var results []*domain.Session
	for rows.Next() {
		var s domain.Session

		err := rows.Scan(
			&s.ID,
			&s.SessionID,
			&s.MSISDN,
			&s.Site,
			&s.PrivateIP,
			&s.PublicIP,
			&s.IPv6,
			&s.StartPort,
			&s.EndPort,
			&s.Packets,
			&s.WLStatus,
			&s.StartTime,
			&s.EndTime,
		)
		if err != nil {
			return nil, fmt.Errorf("session scan: %w", err)
		}

		results = append(results, &s)
	}
	return results, rows.Err()
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}
