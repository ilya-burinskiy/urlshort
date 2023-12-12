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

func Create(
	originalURL string,
	pathLen int,
	randGen RandHexStringGenerator,
	s storage.Storage,
) (models.Record, error) {

	record, err := s.FindByOriginalURL(context.Background(), originalURL)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			shortenedPath, err := randGen.Call(pathLen)
			if err != nil {
				return models.Record{}, fmt.Errorf("failed to generate shortened path: %s", err.Error())
			}

			record = models.Record{OriginalURL: originalURL, ShortenedPath: shortenedPath}
			err = s.Save(context.Background(), record)
			if err != nil {
				return models.Record{}, err
			}

			return record, nil
		}

		return models.Record{}, err
	}

	return models.Record{}, storage.NewErrNotUnique(record)
}

func BatchCreate(
	records []models.Record,
	pahtLen int,
	rndGen RandHexStringGenerator,
	s storage.Storage) error {

	for i := range records {
		shortenedPath, err := rndGen.Call(8)
		if err != nil {
			return fmt.Errorf("failed to generate shortened path for \"%s\": %s",
				records[i].OriginalURL, err.Error())
		}
		records[i].ShortenedPath = shortenedPath
	}

	err := s.BatchSave(context.Background(), records)
	if err != nil {
		return err
	}

	return nil
}
