package repository

import (
	"strings"

	"gorm.io/gorm"
)

// escapeLike escapes LIKE/ILIKE pattern characters so user input is treated literally.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

type Repository struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{DB: db}
}

// GetDB returns the underlying *gorm.DB for cases that need direct access.
func (r *Repository) GetDB() *gorm.DB {
	return r.DB
}

// Transaction runs fn inside a database transaction. Commits when fn returns nil,
// rolls back otherwise.
func (r *Repository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.DB.Transaction(fn)
}
