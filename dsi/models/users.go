package models

import (
	"time"
)

// User is a user of the application
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"-"`
	PasswordHash string    `json:"-"`
	Created      time.Time `json:"created"`
	Tier         string    `json:"app_tier"`
}
