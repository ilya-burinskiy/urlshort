package storage

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")

type Storage interface {
	GetShortenedPath(ctx context.Context, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortenedPath string) (string, error)
	Save(ctx context.Context, originalURL, shortenedPath string) error
}
