package domain

import "time"

type UserType string
type UserStatus bool

const (
	UserTypeAdmin     UserType = "admin"
	UserTypeViewer    UserType = "viewer"
	UserTypeWhitelist UserType = "whitelist"
)

type User struct {
	ID           int
	Username     string
	Email        string // nullable
	PasswordHash string
	UserType     UserType
	Status       bool
	LastLogin    time.Time
}
