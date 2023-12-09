package services

import (
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func StorageDumper(ms storage.MapStorage, fs storage.FileStorage, timeout time.Duration) {
	for {
		fs.Dump(ms)
		time.Sleep(timeout)
	}
}
