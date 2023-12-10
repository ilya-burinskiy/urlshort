package services

import (
	"context"
	"errors"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

type RandHexStringGenerator interface {
	Call(n int) (string, error)
}

func CreateShortenedURLService(
	originalURL,
	shortenedURLBaseAddr string,
	pathLen int,
	randGen RandHexStringGenerator,
	s storage.Storage,
) (string, error) {
	shortenedURLPath, err := s.GetShortenedPath(context.Background(), originalURL)
	if errors.Is(err, storage.ErrNotFound) {
		var err error
		shortenedURLPath, err = randGen.Call(pathLen)
		if err != nil {
			return "", err
		}
		s.Save(
			context.Background(),
			models.Record{OriginalURL: originalURL, ShortenedPath: shortenedURLPath},
		)
	}

	// TODO: maybe use some URL builder
	return shortenedURLBaseAddr + "/" + shortenedURLPath, nil
}
