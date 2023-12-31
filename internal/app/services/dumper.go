package services

import (
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

type StorageDumper struct {
	ms      *storage.MapStorage
	timeout time.Duration
}

func NewStorageDumper(ms *storage.MapStorage, timeout time.Duration) StorageDumper {
	return StorageDumper{
		ms:      ms,
		timeout: timeout,
	}
}

func (d StorageDumper) Start() {
	go func() {
		for {
			d.ms.Dump()
			time.Sleep(d.timeout)
		}
	}()
}
