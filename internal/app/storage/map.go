package storage

import (
	"context"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

type MapStorage map[string]string

func NewMapStorage() MapStorage {
	return MapStorage(make(map[string]string))
}

func (ms MapStorage) GetShortenedPath(ctx context.Context, originalURL string) (string, error) {
	shortenedPath, ok := ms[originalURL]
	if !ok {
		return "", ErrNotFound
	}

	return shortenedPath, nil
}

func (ms MapStorage) GetOriginalURL(ctx context.Context, shortenedPath string) (string, error) {
	for k, v := range ms {
		if v == shortenedPath {
			return k, nil
		}
	}
	return "", ErrNotFound
}

func (ms MapStorage) Save(ctx context.Context, originalURL, shortenedPath string) error {
	ms[originalURL] = shortenedPath
	return nil
}

func (ms MapStorage) BatchSave(ctx context.Context, records []models.Record) error {
	for _, record := range records {
		ms[record.OriginalURL] = record.ShortenedPath
	}
	return nil
}
