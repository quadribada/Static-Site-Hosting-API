package models

import "time"

// Deployment represents a static site deployment
type Deployment struct {
	ID        string    `json:"id" db:"id"`
	Filename  string    `json:"filename" db:"filename"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Path      string    `json:"path" db:"path"`
}

// NewDeployment creates a new deployment instance
func NewDeployment(id, filename, path string) *Deployment {
	return &Deployment{
		ID:        id,
		Filename:  filename,
		Timestamp: time.Now(),
		Path:      path,
	}
}

// TableName returns the database table name for this model
func (d *Deployment) TableName() string {
	return "deployments"
}
