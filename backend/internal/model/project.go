package model

import "time"

// Project is a container for tasks, owned by a single user.
type Project struct {
	ID          string    `db:"id"          json:"id"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description"`
	OwnerID     string    `db:"owner_id"    json:"owner_id"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

// ProjectWithTasks is returned by the GET /projects/:id endpoint.
type ProjectWithTasks struct {
	Project
	Tasks []Task `json:"tasks"`
}
