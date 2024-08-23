package data

import (
	"time"
)

// User is the structure which holds one user from the database.
type User struct {
	ID        string       `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Password  string    `json:"-"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
