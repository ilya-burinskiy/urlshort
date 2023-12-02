package services

import (
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func StorageDumper(s storage.Storage, timeout time.Duration) {
	for {
		s.Dump()
		time.Sleep(timeout)
	}
}
