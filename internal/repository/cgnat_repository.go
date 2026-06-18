package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"taskmanager/internal/domain"
)

type CGNATRepository struct {
	db       *sql.DB
	siteName string
}

func NewCGNATRepository(db *sql.DB, site string) *CGNATRepository {
	return &CGNATRepository{db: db, siteName: site}
}

func (r *CGNATRepository) IsAlive() bool {
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

func (r *CGNATRepository) Create(c *domain.CGNAT, site string) (int64, error) {

	if r.siteName != site {
		return 2, nil
	}

	res, err := r.db.Exec(
		`INSERT INTO cgnat_table
		(private_ip, public_ip, start_port, end_port)
		VALUES (INET6_ATON(?), INET6_ATON(?), ?, ?)`,
		c.PrivateIP,
		c.PublicIP,
		c.StartPort,
		c.EndPort,
	)
	if err != nil {
		return 0, fmt.Errorf("cgnat create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("cgnat create lastInsertId: %w", err)
	}

	return id, nil
}

func (r *CGNATRepository) BulkCreate(entries []*domain.CGNAT, site string) error {

	if r.siteName != site {
		return nil
	}

	if len(entries) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("cgnat bulk create begin tx: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO cgnat_table
		(private_ip, public_ip, start_port, end_port)
		VALUES (INET6_ATON(?), INET6_ATON(?), ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("cgnat bulk create prepare: %w", err)
	}
	defer stmt.Close()

	for _, c := range entries {
		if _, err := stmt.Exec(
			c.PrivateIP,
			c.PublicIP,
			c.StartPort,
			c.EndPort,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("cgnat bulk create exec: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cgnat bulk create commit: %w", err)
	}

	return nil
}

// -------------------- READ (single) --------------------

func (r *CGNATRepository) GetByID(id int64) (*domain.CGNAT, error) {
	row := r.db.QueryRow(
		`SELECT id, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, start_port, end_port
		 FROM cgnat_table
		 WHERE id = ?`,
		id,
	)

	var c domain.CGNAT
	if err := row.Scan(
		&c.ID,
		&c.PrivateIP,
		&c.PublicIP,
		&c.StartPort,
		&c.EndPort,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("cgnat get by id: %w", err)
	}

	return &c, nil
}

// -------------------- LIST --------------------

func (r *CGNATRepository) List() ([]*domain.CGNAT, error) {
	rows, err := r.db.Query(
		`SELECT id, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, start_port, end_port
		 FROM cgnat_table
		 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("cgnat list: %w", err)
	}
	defer rows.Close()

	return scanCGNATRows(rows)
}

// -------------------- FILTERS --------------------

func (r *CGNATRepository) FindByPrivateIP(ip string) ([]*domain.CGNAT, error) {
	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid private IP: %q", ip)
	}

	rows, err := r.db.Query(
		`SELECT id, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, start_port, end_port
		 FROM cgnat_table
		 WHERE private_ip = ?
		 ORDER BY id`,
		ip,
	)
	if err != nil {
		return nil, fmt.Errorf("cgnat find by private_ip: %w", err)
	}
	defer rows.Close()

	return scanCGNATRows(rows)
}

func (r *CGNATRepository) FindByPublicIP(ip string) ([]*domain.CGNAT, error) {
	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid public IP: %q", ip)
	}

	rows, err := r.db.Query(
		`SELECT id, INET6_NTOA(private_ip) AS private_ip, INET6_NTOA(public_ip) AS public_ip, start_port, end_port
		 FROM cgnat_table
		 WHERE public_ip = ?
		 ORDER BY start_port`,
		ip,
	)
	if err != nil {
		return nil, fmt.Errorf("cgnat find by public_ip: %w", err)
	}
	defer rows.Close()

	return scanCGNATRows(rows)
}

// -------------------- DELETE --------------------

func (r *CGNATRepository) Delete(id int64) error {
	_, err := r.db.Exec(
		`DELETE FROM cgnat_table WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("cgnat delete: %w", err)
	}
	return nil
}

// -------------------- HELPERS --------------------

func scanCGNATRows(rows *sql.Rows) ([]*domain.CGNAT, error) {
	var results []*domain.CGNAT
	for rows.Next() {
		var c = domain.NewCGNAT()
		if err := rows.Scan(
			&c.ID,
			&c.PrivateIP,
			&c.PublicIP,
			&c.StartPort,
			&c.EndPort,
		); err != nil {
			return nil, fmt.Errorf("cgnat scan: %w", err)
		}
		results = append(results, &c)
	}
	return results, rows.Err()
}
