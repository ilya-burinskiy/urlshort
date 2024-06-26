package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

// Not found error
var ErrNotFound = errors.New("not found")

// Record not unique error
type ErrNotUnique struct {
	Record models.Record
}

// New record not unique error
func NewErrNotUnique(r models.Record) *ErrNotUnique {
	return &ErrNotUnique{Record: r}
}

// Error
func (err *ErrNotUnique) Error() string {
	return fmt.Sprintf("%v not unique", err.Record)
}

// Storage interface
type Storage interface {
	FindByOriginalURL(ctx context.Context, originalURL string) (models.Record, error)
	FindByShortenedPath(ctx context.Context, shortenedPath string) (models.Record, error)
	FindByUser(ctx context.Context, user models.User) ([]models.Record, error)
	Save(ctx context.Context, record models.Record) error
	BatchSave(ctx context.Context, records []models.Record) error
	BatchDelete(ctx context.Context, records []models.Record) error
	URLsCount(ctx context.Context) (int, error)
	UsersCount(ctx context.Context) (int, error)

	CreateUser(ctx context.Context) (models.User, error)
}
