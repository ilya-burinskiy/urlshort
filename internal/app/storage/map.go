package storage

import (
	"context"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

type MapStorage struct {
	records              []models.Record
	indexOnOriginalURL   map[string]int
	indexOnShortenedPath map[string]int
	indexOnUserID        map[int]map[int]struct{}
	userID               int
	fs                   *FileStorage
}

func NewMapStorage(fs *FileStorage) *MapStorage {
	return &MapStorage{
		records:              make([]models.Record, 0),
		indexOnOriginalURL:   make(map[string]int),
		indexOnShortenedPath: make(map[string]int),
		indexOnUserID:        make(map[int]map[int]struct{}),
		userID:               1,
		fs:                   fs,
	}
}

func (ms *MapStorage) FindByOriginalURL(ctx context.Context, originalURL string) (models.Record, error) {
	idx, ok := ms.indexOnOriginalURL[originalURL]
	if !ok {
		return models.Record{}, ErrNotFound
	}

	return ms.records[idx], nil
}

func (ms *MapStorage) FindByShortenedPath(ctx context.Context, shortenedPath string) (models.Record, error) {
	idx, ok := ms.indexOnShortenedPath[shortenedPath]
	if !ok {
		return models.Record{}, ErrNotFound
	}

	return ms.records[idx], nil
}

func (ms *MapStorage) FindByUser(ctx context.Context, user models.User) ([]models.Record, error) {
	result := make([]models.Record, 0)
	userRecordsIdx, ok := ms.indexOnUserID[user.ID]
	if !ok {
		return result, nil
	}

	for idx := range userRecordsIdx {
		result = append(result, ms.records[idx])
	}

	return result, nil
}

func (ms *MapStorage) Save(ctx context.Context, r models.Record) error {
	_, ok := ms.indexOnOriginalURL[r.OriginalURL]
	if ok {
		return NewErrNotUnique(r)
	}

	ms.records = append(ms.records, r)
	idx := len(ms.records) - 1
	ms.indexOnOriginalURL[r.OriginalURL] = idx
	ms.indexOnShortenedPath[r.ShortenedPath] = idx
	_, ok = ms.indexOnUserID[r.UserID]
	if !ok {
		ms.indexOnUserID[r.UserID] = make(map[int]struct{})
	}
	ms.indexOnUserID[r.UserID][idx] = struct{}{}

	return nil
}

func (ms *MapStorage) BatchSave(ctx context.Context, records []models.Record) error {
	for _, r := range records {
		idx, ok := ms.indexOnOriginalURL[r.OriginalURL]
		if ok {
			oldRecord := ms.records[idx]
			delete(ms.indexOnShortenedPath, oldRecord.ShortenedPath)
			delete(ms.indexOnUserID[oldRecord.UserID], idx)

			ms.records[idx] = r
			ms.indexOnShortenedPath[r.ShortenedPath] = idx
			_, ok = ms.indexOnUserID[r.UserID]
			if !ok {
				ms.indexOnUserID[r.UserID] = make(map[int]struct{})
			}
			ms.indexOnUserID[r.UserID][idx] = struct{}{}
		} else {
			ms.records = append(ms.records, r)
			idx = len(ms.records) - 1
			ms.indexOnOriginalURL[r.OriginalURL] = idx
			ms.indexOnShortenedPath[r.ShortenedPath] = idx
			_, ok := ms.indexOnUserID[r.UserID]
			if !ok {
				ms.indexOnUserID[r.UserID] = make(map[int]struct{})
			}
			ms.indexOnUserID[r.UserID][idx] = struct{}{}
		}
	}
	return nil
}

func (ms *MapStorage) BatchDelete(ctx context.Context, records []models.Record) error {
	for _, r := range records {
		idx, ok := ms.indexOnShortenedPath[r.ShortenedPath]
		if !ok {
			continue
		}

		usersRecords, ok := ms.indexOnUserID[r.UserID]
		if !ok {
			continue
		}
		if _, ok = usersRecords[idx]; !ok {
			continue
		}

		ms.records[idx].IsDeleted = true
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
		return ms.fs.Dump(ms)
	}

	return nil
}

func (ms *MapStorage) Restore(records []models.Record) {
	ctx := context.TODO()
	maxUserID := 0
	for _, r := range records {
		if r.UserID > maxUserID {
			maxUserID = r.UserID
		}
		ms.Save(ctx, r)
	}
	ms.userID = maxUserID + 1
}
