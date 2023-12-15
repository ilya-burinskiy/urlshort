package services

import (
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func StorageDumper(ms storage.MapStorage, timeout time.Duration) {
	for {
		ms.Dump()
		time.Sleep(timeout)
	}
}
