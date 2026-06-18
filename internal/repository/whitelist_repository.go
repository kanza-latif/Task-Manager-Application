package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"taskmanager/internal/domain"
)

type WhitelistRepository struct {
	db *sql.DB
}

func NewWhitelistRepository(db *sql.DB) *WhitelistRepository {
	return &WhitelistRepository{db: db}
}

func (r *WhitelistRepository) IsAlive() bool {
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

func (r *WhitelistRepository) Create(w *domain.Whitelist) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO whitelist_table (msisdn) VALUES (?)`,
		w.MSISDN,
	)
	if err != nil {
		return 0, fmt.Errorf("whitelist create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("whitelist create lastInsertId: %w", err)
	}

	return id, nil
}

func (r *WhitelistRepository) BulkCreate(entries []*domain.Whitelist) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("whitelist bulk create begin tx: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO whitelist_table (msisdn)
		VALUES (?)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("whitelist bulk create prepare: %w", err)
	}
	defer stmt.Close()

	for _, w := range entries {
		if w.MSISDN == "" {
			continue
		}

		_, err := stmt.Exec(w.MSISDN)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("whitelist bulk create exec: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("whitelist bulk create commit: %w", err)
	}

	return nil
}

// -------------------- READ (single) --------------------

func (r *WhitelistRepository) GetByID(id int64) (*domain.Whitelist, error) {
	row := r.db.QueryRow(
		`SELECT id, msisdn
		 FROM whitelist_table
		 WHERE id = ?`,
		id,
	)

	var w domain.Whitelist
	if err := row.Scan(&w.ID, &w.MSISDN); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("whitelist get by id: %w", err)
	}

	return &w, nil
}

// -------------------- LIST --------------------

func (r *WhitelistRepository) List() ([]*domain.Whitelist, error) {
	rows, err := r.db.Query(
		`SELECT id, msisdn
		 FROM whitelist_table
		 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("whitelist list: %w", err)
	}
	defer rows.Close()

	return scanWhitelistRows(rows)
}

// -------------------- FIND BY MSISDN --------------------

func (r *WhitelistRepository) FindByMSISDN(msisdn string) (*domain.Whitelist, error) {
	row := r.db.QueryRow(
		`SELECT id, msisdn
		 FROM whitelist_table
		 WHERE msisdn = ?`,
		msisdn,
	)

	var w domain.Whitelist
	if err := row.Scan(&w.ID, &w.MSISDN); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("whitelist find by msisdn: %w", err)
	}

	return &w, nil
}

// -------------------- DELETE --------------------

func (r *WhitelistRepository) Delete(id int64) error {
	_, err := r.db.Exec(
		`DELETE FROM whitelist_table WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("whitelist delete: %w", err)
	}
	return nil
}

// -------------------- HELPERS --------------------

func scanWhitelistRows(rows *sql.Rows) ([]*domain.Whitelist, error) {
	var results []*domain.Whitelist

	for rows.Next() {
		var w = domain.NewWhitelist()

		if err := rows.Scan(&w.ID, &w.MSISDN); err != nil {
			return nil, fmt.Errorf("whitelist scan: %w", err)
		}

		results = append(results, &w)
	}

	return results, rows.Err()
}
