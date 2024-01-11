package services

import (
	"context"
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"go.uber.org/zap"
)

type BatchDeleter struct {
	store storage.Storage
	ch    chan models.Record
}

func NewBatchDeleter(store storage.Storage) *BatchDeleter {
	return &BatchDeleter{store: store, ch: make(chan models.Record, 1024)}
}

func (d *BatchDeleter) Delete(record models.Record) {
	d.ch <- record
}

func (d *BatchDeleter) Run() {
	ticker := time.NewTicker(5 * time.Second)
	var records []models.Record

	for {
		select {
		case record := <-d.ch:
			records = append(records, record)
		case <-ticker.C:
			if len(records) == 0 {
				continue
			}

			err := d.store.BatchDelete(context.TODO(), records)
			if err != nil {
				logger.Log.Info("run batch delete error", zap.String("err", err.Error()))
				continue
			}
			records = nil
		}
	}
}
