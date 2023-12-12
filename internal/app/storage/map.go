package storage

import (
	"context"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

type MapStorage map[string][2]string

func NewMapStorage() MapStorage {
	return MapStorage(make(map[string][2]string))
}

func (ms MapStorage) FindByOriginalURL(ctx context.Context, originalURL string) (models.Record, error) {
	attrs, ok := ms[originalURL]
	if !ok {
		return models.Record{}, ErrNotFound
	}

	return models.Record{
		OriginalURL:   originalURL,
		ShortenedPath: attrs[0],
		CorrelationID: attrs[1],
	}, nil
}

func (ms MapStorage) FindByShortenedPath(ctx context.Context, searchedShortenedPath string) (models.Record, error) {
	for originalURL, vals := range ms {
		if vals[0] == searchedShortenedPath {
			return models.Record{
				OriginalURL:   originalURL,
				ShortenedPath: vals[0],
				CorrelationID: vals[1],
			}, nil
		}
	}
	return models.Record{}, ErrNotFound
}

func (ms MapStorage) Save(ctx context.Context, r models.Record) error {
	_, ok := ms[r.OriginalURL]
	if ok {
		return NewErrNotUnique(r)
	}

	ms[r.OriginalURL] = [2]string{r.ShortenedPath, r.CorrelationID}
	return nil
}

func (ms MapStorage) BatchSave(ctx context.Context, records []models.Record) error {
	for _, record := range records {
		ms[record.OriginalURL] = [2]string{record.ShortenedPath, record.CorrelationID}
	}
	return nil
}
