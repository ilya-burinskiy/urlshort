package services

import (
	"context"
	"fmt"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)


// Interface for a hex string generation
type HexStrGen interface {
	Gen(n int) (string, error)
}

// Interface for creating shortened URLs
type URLShortener interface {
	Shortify(string, models.User) (models.Record, error)
	BatchShortify([]models.Record, models.User) ([]models.Record, error)
}

type URLSaver interface {
	Save(ctx context.Context, record models.Record) error
	BatchSave(ctx context.Context, records []models.Record) error
}

type urlShortener struct {
	strGen  HexStrGen 
	urlSaver   URLSaver
	pathLen int
}

// NewURLShortener
func NewURLShortener(pathLen int, strGen HexStrGen, urlSaver URLSaver) URLShortener {
	return urlShortener{
		pathLen: pathLen,
		strGen: strGen,
		urlSaver:   urlSaver,
	}
}

// Create
func (srv urlShortener) Shortify(originalURL string, user models.User) (models.Record, error) {
	shortenedPath, err := srv.strGen.Gen(srv.pathLen)
	if err != nil {
		return models.Record{}, fmt.Errorf("failed to generate shortened path: %s", err.Error())
	}

	record := models.Record{OriginalURL: originalURL, ShortenedPath: shortenedPath, UserID: user.ID}
	err = srv.urlSaver.Save(context.Background(), record)
	if err != nil {
		return models.Record{}, fmt.Errorf("failed to generate shortened path: %s", err.Error())
	}

	return record, nil
}

// BatchCreate
func (srv urlShortener) BatchShortify(records []models.Record, user models.User) ([]models.Record, error) {
	for i := range records {
		shortenedPath, err := srv.strGen.Gen(srv.pathLen)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to generate shortened path for \"%s\": %s",
				records[i].OriginalURL, err.Error(),
			)
		}
		records[i].ShortenedPath = shortenedPath
		records[i].UserID = user.ID
	}

	err := srv.urlSaver.BatchSave(context.Background(), records)
	if err != nil {
		return nil, err
	}

	return records, nil
}
