package services

import "github.com/ilya-burinskiy/urlshort/internal/app/storage"

type RandHexStringGenerator interface {
	Call(n int) (string, error)
}

func CreateShortenedURLService(
	originalURL,
	shortenedURLBaseAddr string,
	pathLen int,
	randGen RandHexStringGenerator,
	storage storage.MapStorage,
) (string, error) {
	shortenedURLPath, ok := storage.Get(originalURL)
	if !ok {
		var err error
		shortenedURLPath, err = randGen.Call(pathLen)
		if err != nil {
			return "", err
		}
		storage.Put(originalURL, shortenedURLPath)
	}

	// TODO: maybe use some URL builder
	return shortenedURLBaseAddr + "/" + shortenedURLPath, nil
}
