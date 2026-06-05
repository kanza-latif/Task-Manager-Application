package repository

import (
	"database/sql"
	"fmt"
	"taskmanager/internal/db/models"
)

type WhitelistRepository struct {
	db *sql.DB
}

func NewWhitelistRepository(db *sql.DB) *WhitelistRepository {
	return &WhitelistRepository{db: db}
}

// List returns all MSISDNs in whitelist_table ordered by id.
func (r *WhitelistRepository) List() ([]*models.Whitelist, error) {
	rows, err := r.db.Query(
		`SELECT id, msisdn FROM whitelist_table ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("whitelist list: %w", err)
	}
	defer rows.Close()

	var results []*models.Whitelist
	for rows.Next() {
		w := &models.Whitelist{}
		if err := rows.Scan(&w.ID, &w.MSISDN); err != nil {
			return nil, fmt.Errorf("whitelist scan: %w", err)
		}
		results = append(results, w)
	}
	return results, rows.Err()
}

// Check returns true if the given MSISDN exists in whitelist_table.
// Uses the unique index on msisdn for an efficient point lookup.
func (r *WhitelistRepository) Check(msisdn string) (bool, error) {
	if msisdn == "" {
		return false, fmt.Errorf("msisdn cannot be empty")
	}
	var id int64
	err := r.db.QueryRow(
		`SELECT id FROM whitelist_table WHERE msisdn = ? LIMIT 1`, msisdn,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("whitelist check: %w", err)
	}
	return true, nil
}

// Count returns total number of whitelisted MSISDNs.
func (r *WhitelistRepository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM whitelist_table`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("whitelist count: %w", err)
	}
	return count, nil
}
