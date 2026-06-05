package repository

import (
	"database/sql"
	"fmt"
	"taskmanager/internal/db/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// List returns all users from user_table.
func (r *UserRepository) List() ([]*models.User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, COALESCE(email, ''), password_hash, user_type, status, last_login
		 FROM user_table ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("user list: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
			&u.UserType, &u.Status, &u.LastLogin); err != nil {
			return nil, fmt.Errorf("user scan: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetByUsername fetches a single user by username.
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, username, COALESCE(email, ''), password_hash, user_type, status, last_login
		 FROM user_table WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.UserType, &u.Status, &u.LastLogin)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user '%s' not found", username)
	}
	if err != nil {
		return nil, fmt.Errorf("user get: %w", err)
	}
	return u, nil
}

// ListByType returns users filtered by user_type.
func (r *UserRepository) ListByType(userType string) ([]*models.User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, COALESCE(email, ''), password_hash, user_type, status, last_login
		 FROM user_table WHERE user_type = ? ORDER BY id`, userType,
	)
	if err != nil {
		return nil, fmt.Errorf("user list by type: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
			&u.UserType, &u.Status, &u.LastLogin); err != nil {
			return nil, fmt.Errorf("user scan: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
