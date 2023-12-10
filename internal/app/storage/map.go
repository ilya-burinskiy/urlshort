package storage

import (
	"context"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

type MapStorage map[string][2]string

func NewMapStorage() MapStorage {
	return MapStorage(make(map[string][2]string))
}

func (ms MapStorage) GetShortenedPath(ctx context.Context, originalURL string) (string, error) {
	attrs, ok := ms[originalURL]
	if !ok {
		return "", ErrNotFound
	}
	shortenedPath := attrs[0]

	return shortenedPath, nil
}

func (ms MapStorage) GetOriginalURL(ctx context.Context, searchedShortenedPath string) (string, error) {
	for originalURL, vals := range ms {
		shortenedPath := vals[0]
		if shortenedPath == searchedShortenedPath {
			return originalURL, nil
		}
	}
	return "", ErrNotFound
}

func (ms MapStorage) Save(ctx context.Context, r models.Record) error {
	ms[r.OriginalURL] = [2]string{r.ShortenedPath, r.CorrelationID}
	return nil
}

func (ms MapStorage) BatchSave(ctx context.Context, records []models.Record) error {
	for _, record := range records {
		ms[record.OriginalURL] = [2]string{record.ShortenedPath, record.CorrelationID}
	}
	return nil
}
