package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"taskmanager/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) IsAlive() bool {
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

func (r *UserRepository) Create(u *domain.User) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO user_table
		(username, email, password_hash, user_type, status, last_login)
		VALUES (?, ?, ?, ?, ?, ?)`,
		u.Username,
		u.Email,
		u.PasswordHash,
		u.UserType,
		u.Status,
		u.LastLogin,
	)
	if err != nil {
		return 0, fmt.Errorf("user create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("user create lastInsertId: %w", err)
	}

	return id, nil
}

// -------------------- READ (single) --------------------

func (r *UserRepository) GetByID(id int) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, email, password_hash, user_type, status, last_login
		 FROM user_table
		 WHERE id = ?`,
		id,
	)

	var u domain.User
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.UserType,
		&u.Status,
		&u.LastLogin,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("user get by id: %w", err)
	}

	return &u, nil
}

// -------------------- LIST --------------------

func (r *UserRepository) List() ([]*domain.User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, email, password_hash, user_type, status, last_login
		 FROM user_table
		 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("user list: %w", err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

// -------------------- FILTERS --------------------

func (r *UserRepository) FindByUsername(username string) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, email, password_hash, user_type, status, last_login
		 FROM user_table
		 WHERE username = ?`,
		username,
	)

	var u domain.User
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.UserType,
		&u.Status,
		&u.LastLogin,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("user find by username: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, email, password_hash, user_type, status, last_login
		 FROM user_table
		 WHERE email = ?`,
		email,
	)

	var u domain.User
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.UserType,
		&u.Status,
		&u.LastLogin,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("user find by email: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) FindByType(userType domain.UserType) ([]*domain.User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, email, password_hash, user_type, status, last_login
		 FROM user_table
		 WHERE user_type = ?
		 ORDER BY id`,
		userType,
	)
	if err != nil {
		return nil, fmt.Errorf("user find by type: %w", err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

func (r *UserRepository) FindByStatus(status bool) ([]*domain.User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, email, password_hash, user_type, status, last_login
		 FROM user_table
		 WHERE status = ?
		 ORDER BY id`,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("user find by status: %w", err)
	}
	defer rows.Close()

	return scanUserRows(rows)
}

// -------------------- UPDATE --------------------

func (r *UserRepository) Update(u *domain.User) error {
	_, err := r.db.Exec(
		`UPDATE user_table SET
			username = ?,
			email = ?,
			password_hash = ?,
			user_type = ?,
			status = ?,
			last_login = ?
		WHERE id = ?`,
		u.Username,
		u.Email,
		u.PasswordHash,
		u.UserType,
		u.Status,
		u.LastLogin,
		u.ID,
	)

	if err != nil {
		return fmt.Errorf("user update: %w", err)
	}

	return nil
}

// -------------------- DELETE --------------------

func (r *UserRepository) Delete(id int) error {
	_, err := r.db.Exec(
		`DELETE FROM user_table WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("user delete: %w", err)
	}
	return nil
}

// -------------------- HELPERS --------------------

func scanUserRows(rows *sql.Rows) ([]*domain.User, error) {
	var results []*domain.User

	for rows.Next() {
		var u domain.User

		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.PasswordHash,
			&u.UserType,
			&u.Status,
			&u.LastLogin,
		); err != nil {
			return nil, fmt.Errorf("user scan: %w", err)
		}

		results = append(results, &u)
	}

	return results, rows.Err()
}
