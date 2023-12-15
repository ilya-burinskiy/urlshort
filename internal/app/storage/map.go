package storage

import (
	"context"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

type MapStorage map[string]link

type link struct {
	ShortenedPath string `json:"shortened_path"`
	CorrelationID string `json:"correlation_id"`
}

func NewMapStorage() MapStorage {
	return MapStorage(make(map[string]link))
}

func (ms MapStorage) FindByOriginalURL(ctx context.Context, originalURL string) (models.Record, error) {
	l, ok := ms[originalURL]
	if !ok {
		return models.Record{}, ErrNotFound
	}

	return models.Record{
		OriginalURL:   originalURL,
		ShortenedPath: l.ShortenedPath,
		CorrelationID: l.CorrelationID,
	}, nil
}

func (ms MapStorage) FindByShortenedPath(ctx context.Context, searchedShortenedPath string) (models.Record, error) {
	for originalURL, l := range ms {
		if l.ShortenedPath == searchedShortenedPath {
			return models.Record{
				OriginalURL:   originalURL,
				ShortenedPath: l.ShortenedPath,
				CorrelationID: l.CorrelationID,
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

	ms[r.OriginalURL] = link{
		ShortenedPath: r.ShortenedPath,
		CorrelationID: r.CorrelationID,
	}
	return nil
}

func (ms MapStorage) BatchSave(ctx context.Context, records []models.Record) error {
	for _, record := range records {
		ms[record.OriginalURL] = link{
			ShortenedPath: record.ShortenedPath,
			CorrelationID: record.CorrelationID,
		}
	}
	return nil
}
