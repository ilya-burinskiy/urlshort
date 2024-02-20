package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

type RandHexStringGenerator interface {
	Call(n int) (string, error)
}

type CreateURLService interface {
	Create(string, models.User) (models.Record, error)
	BatchCreate([]models.Record, models.User) error
}

type createURLService struct {
	pathLen int
	randGen RandHexStringGenerator
	store   storage.Storage
}

func NewCreateURLService(pathLen int, randGen RandHexStringGenerator, store storage.Storage) CreateURLService {
	return createURLService{
		pathLen: pathLen,
		randGen: randGen,
		store:   store,
	}
}

func (service createURLService) Create(originalURL string, user models.User) (models.Record, error) {
	record, err := service.store.FindByOriginalURL(context.Background(), originalURL)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			shortenedPath, err := service.randGen.Call(service.pathLen)
			if err != nil {
				return models.Record{}, fmt.Errorf("failed to generate shortened path: %s", err.Error())
			}

			record = models.Record{OriginalURL: originalURL, ShortenedPath: shortenedPath, UserID: user.ID}
			err = service.store.Save(context.Background(), record)
			if err != nil {
				return models.Record{}, err
			}

			return record, nil
		}

		return models.Record{}, err
	}

	return models.Record{}, storage.NewErrNotUnique(record)
}

func (service createURLService) BatchCreate(records []models.Record, user models.User) error {
	for i := range records {
		shortenedPath, err := service.randGen.Call(service.pathLen)
		if err != nil {
			return fmt.Errorf("failed to generate shortened path for \"%s\": %s",
				records[i].OriginalURL, err.Error())
		}
		records[i].ShortenedPath = shortenedPath
		records[i].UserID = user.ID
	}

	err := service.store.BatchSave(context.Background(), records)
	if err != nil {
		return err
	}

	return nil
}
