package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

type FileStorage struct {
	filePath string
}

func NewFileStorage(filePath string) *FileStorage {
	return &FileStorage{filePath: filePath}
}

func (fs *FileStorage) Restore(ms MapStorage) error {
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not load data from file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var r models.Record
		data := scanner.Bytes()
		err = json.Unmarshal(data, &r)
		if err != nil {
			continue
		}

		ms.Save(context.Background(), r)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("could not restore data: %s", err.Error())
	}

	return scanner.Err()
}

func (fs *FileStorage) Dump(ms MapStorage) error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("could not dump storage: %w", err)
	}

	encoder := json.NewEncoder(file)
	// NOTE: maybe define some Iter method for MapStorage
	for k, l := range ms {
		shortenedPath := l.ShortenedPath
		correlationID := l.CorrelationID
		encoder.Encode(models.Record{OriginalURL: k, ShortenedPath: shortenedPath, CorrelationID: correlationID})
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("could not dump storage: %w", err)
	}

	return nil
}
