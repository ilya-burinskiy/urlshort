package storage

import (
	"context"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

type MapStorage struct {
	m      map[string]link
	userID int
	fs     *FileStorage
}

type link struct {
	ShortenedPath string `json:"shortened_path"`
	CorrelationID string `json:"correlation_id"`
	UserID        int    `json:"user_id"`
	IsDeleted     bool   `json:"is_deleted"`
}

func NewMapStorage(fs *FileStorage) *MapStorage {
	return &MapStorage{
		m:      make(map[string]link),
		userID: 1,
		fs:     fs,
	}
}

func (ms *MapStorage) FindByOriginalURL(ctx context.Context, originalURL string) (models.Record, error) {
	l, ok := ms.m[originalURL]
	if !ok {
		return models.Record{}, ErrNotFound
	}

	return models.Record{
		OriginalURL:   originalURL,
		ShortenedPath: l.ShortenedPath,
		CorrelationID: l.CorrelationID,
	}, nil
}

func (ms *MapStorage) FindByShortenedPath(ctx context.Context, searchedShortenedPath string) (models.Record, error) {
	for originalURL, l := range ms.m {
		if l.ShortenedPath == searchedShortenedPath {
			return models.Record{
				OriginalURL:   originalURL,
				ShortenedPath: l.ShortenedPath,
				CorrelationID: l.CorrelationID,
				IsDeleted:     l.IsDeleted,
			}, nil
		}
	}
	return models.Record{}, ErrNotFound
}

func (ms *MapStorage) FindByUser(ctx context.Context, user models.User) ([]models.Record, error) {
	result := make([]models.Record, 0)
	for origURL, l := range ms.m {
		if l.UserID == user.ID {
			result = append(result, models.Record{
				OriginalURL:   origURL,
				ShortenedPath: l.ShortenedPath,
				CorrelationID: l.CorrelationID,
				UserID:        l.UserID,
			})
		}
	}

	return result, nil
}

func (ms *MapStorage) Save(ctx context.Context, r models.Record) error {
	_, ok := ms.m[r.OriginalURL]
	if ok {
		return NewErrNotUnique(r)
	}

	ms.m[r.OriginalURL] = link{
		ShortenedPath: r.ShortenedPath,
		CorrelationID: r.CorrelationID,
		UserID:        r.UserID,
	}
	return nil
}

func (ms *MapStorage) BatchSave(ctx context.Context, records []models.Record) error {
	for _, record := range records {
		ms.m[record.OriginalURL] = link{
			ShortenedPath: record.ShortenedPath,
			CorrelationID: record.CorrelationID,
			UserID:        record.UserID,
		}
	}
	return nil
}

func (ms *MapStorage) CreateUser(ctx context.Context) (models.User, error) {
	id := ms.userID
	ms.userID++

	return models.User{ID: id}, nil
}

func (ms *MapStorage) Dump() error {
	if ms.fs != nil {
		return ms.fs.Dump(*ms)
	}

	return nil
}

func (ms *MapStorage) Restore() error {
	if ms.fs != nil {
		return ms.fs.Restore(ms)
	}

	return nil
}
