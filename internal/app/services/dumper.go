package services

import (
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"go.uber.org/zap"
)

// StorageDumber
type StorageDumper struct {
	ms      *storage.MapStorage
	timeout time.Duration
}

// NewStorageDumper
func NewStorageDumper(ms *storage.MapStorage, timeout time.Duration) StorageDumper {
	return StorageDumper{
		ms:      ms,
		timeout: timeout,
	}
}

// Start
func (d StorageDumper) Start() {
	go func() {
		for {
			if err := d.ms.Dump(); err != nil {
				logger.Log.Info("storage dumper", zap.Error(err))
			}
			time.Sleep(d.timeout)
		}
	}()
}
