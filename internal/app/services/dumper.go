package services

import (
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
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
			d.ms.Dump()
			time.Sleep(d.timeout)
		}
	}()
}
