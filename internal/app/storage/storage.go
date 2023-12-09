package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type MapStorage struct {
	m                 map[string]string
	persistentStorage PersistentStorage
}

func NewMapStorage(ps PersistentStorage) MapStorage {
	return MapStorage{m: make(map[string]string), persistentStorage: ps}
}

func (ms MapStorage) Put(key, val string) {
	ms.m[key] = val
	ms.Dump()
}

func (ms MapStorage) Get(key string) (string, bool) {
	val, ok := ms.m[key]
	return val, ok
}

func (ms MapStorage) Clear() {
	for k := range ms.m {
		delete(ms.m, k)
	}
}

func (ms MapStorage) KeyByValue(value string) (string, bool) {
	for k, v := range ms.m {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func (ms MapStorage) Dump() error {
	if ms.persistentStorage != nil {
		return ms.persistentStorage.Dump(ms)
	}
	return nil
}

func (ms MapStorage) Restore() error {
	if ms.persistentStorage != nil {
		return ms.persistentStorage.Restore(ms)
	}
	return nil
}

type PersistentStorage interface {
	Dump(ms MapStorage) error
	Restore(ms MapStorage) error
}

func ConfigurePersistentStorage(config configs.Config) PersistentStorage {
	if config.DatabaseDSN != "" {
		return NewDBStorage(config.DatabaseDSN)
	}
	if config.FileStoragePath != "" {
		return NewFileStorage(config.FileStoragePath)
	}
	return nil
}

type FileStorage struct {
	filePath string
}

type record struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func NewFileStorage(filePath string) FileStorage {
	return FileStorage{filePath: filePath}
}

func (fs FileStorage) Restore(ms MapStorage) error {
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not load data from file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var r record
		data := scanner.Bytes()
		err = json.Unmarshal(data, &r)
		if err != nil {
			continue
		}

		ms.Put(r.Key, r.Val)
	}

	return scanner.Err()
}

func (fs FileStorage) Dump(ms MapStorage) error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("could not dump storage: %s", err)
	}

	encoder := json.NewEncoder(file)
	// NOTE: maybe define some Iter method for MapStorage
	for k, v := range ms.m {
		encoder.Encode(record{Key: k, Val: v})
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("could not dump storage: %s", err)
	}

	return nil
}

type DBStorage struct {
	dsn string
}

func NewDBStorage(dsn string) DBStorage {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		logger.Log.Info("could not migrate db", zap.String("msg", err.Error()))
		return DBStorage{dsn: dsn}
	}
	migrateQuery := `
		CREATE TABLE IF NOT EXISTS "urls" (
			"id" bigserial PRIMARY KEY,
			"original_url" varchar(499) UNIQUE NOT NULL,
			"shortened_path" varchar(499) UNIQUE NOT NULL
		)`
	_, err = conn.Exec(context.Background(), migrateQuery)
	if err != nil {
		logger.Log.Info("could not migrate db", zap.String("msg", err.Error()))
		return DBStorage{dsn: dsn}
	}

	return DBStorage{dsn: dsn}
}

func (ds DBStorage) Restore(ms MapStorage) error {
	conn, err := pgx.Connect(context.Background(), ds.dsn)
	if err != nil {
		return fmt.Errorf("could not restore data from db: %s", err)
	}
	defer conn.Close(context.Background())

	var originalURL, shortenedPath string
	rows, err := conn.Query(context.Background(), `SELECT "original_url", "shortened_path" FROM "urls"`)
	if err != nil {
		logger.Log.Info("could not exec restore query", zap.String("msg", err.Error()))
	}
	for rows.Next() {
		err = rows.Scan(&originalURL, &shortenedPath)
		if err != nil {
			logger.Log.Info("could not scan urls", zap.String("msg", err.Error()))
			continue
		}
		ms.Put(originalURL, shortenedPath)
	}

	return nil
}

func (ds DBStorage) Dump(ms MapStorage) error {
	conn, err := pgx.Connect(context.Background(), ds.dsn)
	if err != nil {
		return fmt.Errorf("could not restore data from db: %s", err)
	}
	defer conn.Close(context.Background())

	query := `
		INSERT INTO "urls" ("original_url", "shortened_path") VALUES (@originalURL, @shortenedPath)
		ON CONFLICT DO NOTHING`
	for originalURL, shortenedPath := range ms.m {
		queryArgs := pgx.NamedArgs{
			"originalURL":   originalURL,
			"shortenedPath": shortenedPath,
		}
		_, err = conn.Exec(context.Background(), query, queryArgs)
		if err != nil {
			logger.Log.Info("could not insert urls", zap.String("msg", err.Error()))
			continue
		}
	}

	return nil
}
