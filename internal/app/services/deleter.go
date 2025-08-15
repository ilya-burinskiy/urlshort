package services

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

// BatchDeleter
type BatchDeleter interface {
	BatchDelete(ctx context.Context, records []models.Record) error
}

type DeferredDeleter struct {
	batchDeleter BatchDeleter
	ch chan models.Record
}

func NewDeferredDeleter(batchDeleter BatchDeleter) DeferredDeleter {
	return DeferredDeleter{batchDeleter: batchDeleter, ch: make(chan models.Record, 1024)}
}

func (d DeferredDeleter) Enqueue(record models.Record) {
	d.ch <- record
}

// Run
func (d DeferredDeleter) Run() {
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

			err := d.batchDeleter.BatchDelete(context.TODO(), records)
			if err != nil {
				logger.Log.Info("run batch delete error", zap.String("err", err.Error()))
				continue
			}
			records = nil
		}
	}
}
