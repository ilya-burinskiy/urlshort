package storage

import (
	"context"
	"errors"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

var ErrNotFound = errors.New("not found")

type Storage interface {
	GetShortenedPath(ctx context.Context, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortenedPath string) (string, error)
	Save(ctx context.Context, record models.Record) error
	BatchSave(ctx context.Context, records []models.Record) error
}
