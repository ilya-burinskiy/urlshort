package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

// File storage
type FileStorage struct {
	filePath string
}

// New file storage
func NewFileStorage(filePath string) *FileStorage {
	return &FileStorage{filePath: filePath}
}

// Get records from file
func (fs *FileStorage) Snapshot() ([]models.Record, error) {
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("could not load data from file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	result := make([]models.Record, 0)
	for scanner.Scan() {
		var r models.Record
		data := scanner.Bytes()
		err = json.Unmarshal(data, &r)
		if err != nil {
			continue
		}
		result = append(result, r)
	}

	err = file.Close()
	if err != nil {
		return nil, fmt.Errorf("could not restore data: %s", err.Error())
	}

	return result, scanner.Err()

}

// Save records to file
func (fs *FileStorage) Dump(ms *MapStorage) error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("could not dump storage: %w", err)
	}

	encoder := json.NewEncoder(file)
	// NOTE: maybe define some Iter method for MapStorage
	for _, r := range ms.records {
		encoder.Encode(r)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("could not dump storage: %w", err)
	}

	return nil
}
