package model

import "time"

// User represents an authenticated principal in the system.
type User struct {
	ID        string    `db:"id"         json:"id"`
	Name      string    `db:"name"        json:"name"`
	Email     string    `db:"email"       json:"email"`
	Password  string    `db:"password"    json:"-"` // bcrypt hash — never serialized
	CreatedAt time.Time `db:"created_at"  json:"created_at"`
}
