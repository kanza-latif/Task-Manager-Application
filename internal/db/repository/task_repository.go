package repository

import (
	"database/sql"
	"fmt"
	"taskmanager/internal/db/models"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(title, description string) (*models.Task, error) {
	res, err := r.db.Exec(
		`INSERT INTO tasks (title, description, status) VALUES (?, ?, 'todo')`,
		title, description,
	)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}
	return r.GetByID(int(id))
}

func (r *TaskRepository) GetByID(id int) (*models.Task, error) {
	row := r.db.QueryRow(
		`SELECT id, title, description, status, created_at, updated_at FROM tasks WHERE id = ?`, id,
	)
	return scanTask(row)
}

func (r *TaskRepository) List(filter string) ([]*models.Task, error) {
	query := `SELECT id, title, description, status, created_at, updated_at FROM tasks`
	args := []any{}

	if filter == "todo" || filter == "done" {
		query += ` WHERE status = ?`
		args = append(args, filter)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		t := &models.Task{}
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *TaskRepository) Update(id int, title, description string) (*models.Task, error) {
	existing, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if title != "" {
		existing.Title = title
	}
	if description != "" {
		existing.Description = description
	}
	_, err = r.db.Exec(
		`UPDATE tasks SET title = ?, description = ? WHERE id = ?`,
		existing.Title, existing.Description, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return r.GetByID(id)
}

func (r *TaskRepository) MarkDone(id int) error {
	res, err := r.db.Exec(`UPDATE tasks SET status = 'done' WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("mark done: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task %d not found", id)
	}
	return nil
}

func (r *TaskRepository) Delete(id int) error {
	res, err := r.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task %d not found", id)
	}
	return nil
}

func scanTask(row *sql.Row) (*models.Task, error) {
	t := &models.Task{}
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan task: %w", err)
	}
	return t, nil
}
