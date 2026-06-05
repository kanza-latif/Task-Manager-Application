package models

import "time"

type Status string

const (
	StatusTodo Status = "todo"
	StatusDone Status = "done"
)

type Task struct {
	ID          int
	Title       string
	Description string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
