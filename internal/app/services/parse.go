package services

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

func ParseBatchRecods(r io.Reader) ([]models.Record, error) {
	result := make([]models.Record, 0)
	err := json.NewDecoder(r).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse records: %w", err)
	}
	return result, nil
}
