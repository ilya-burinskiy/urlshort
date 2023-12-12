package storage

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("failed to run DB migrations: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create a connection pool: %w", err)
	}

	return &DBStorage{
		pool: pool,
	}, nil
}

func (db *DBStorage) FindByOriginalURL(ctx context.Context, originalURL string) (models.Record, error) {
	row := db.pool.QueryRow(
		ctx,
		`SELECT "shortened_path",
				"correlation_id"
		 FROM "urls" WHERE "original_url" = @originalUrl`,
		pgx.NamedArgs{"originalUrl": originalURL},
	)
	var shortenedPath, correlationID string
	err := row.Scan(&shortenedPath, &correlationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Record{}, ErrNotFound
		}

		return models.Record{}, fmt.Errorf("failed to get shortened path: %w", err)
	}

	return models.Record{
		OriginalURL:   originalURL,
		ShortenedPath: shortenedPath,
		CorrelationID: correlationID,
	}, nil
}

func (db *DBStorage) FindByShortenedPath(ctx context.Context, shortenedPath string) (models.Record, error) {
	row := db.pool.QueryRow(
		ctx,
		`SELECT "original_url",
				"correlation_id"
		 FROM "urls" WHERE "shortened_path" = @shortenedPath`,
		pgx.NamedArgs{"shortenedPath": shortenedPath},
	)
	var originalURL, correlationID string
	err := row.Scan(&originalURL, &correlationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Record{}, ErrNotFound
		}

		return models.Record{}, fmt.Errorf("failed to get original url: %w", err)
	}

	return models.Record{
		OriginalURL:   originalURL,
		ShortenedPath: shortenedPath,
		CorrelationID: correlationID,
	}, nil
}

func (db *DBStorage) Save(ctx context.Context, record models.Record) error {
	_, err := db.pool.Exec(
		ctx,
		`INSERT INTO "urls" ("original_url", "shortened_path") VALUES (@originalURL, @shortenedPath)`,
		pgx.NamedArgs{"originalURL": record.OriginalURL, "shortenedPath": record.ShortenedPath},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return NewErrNotUnique(record)
			}
		}
		return fmt.Errorf("failed to save original url and shortened path: %w", err)
	}

	return nil
}

func (db *DBStorage) BatchSave(ctx context.Context, records []models.Record) error {
	for _, r := range records {
		_, err := db.pool.Exec(
			ctx,
			`INSERT INTO "urls" ("original_url", "shortened_path") VALUES (@originalURL, @shortenedPath)
			 ON CONFLICT ("original_url") DO UPDATE SET "shortened_path" = @shortenedPath`,
			pgx.NamedArgs{"originalURL": r.OriginalURL, "shortenedPath": r.ShortenedPath},
		)
		if err != nil {
			return fmt.Errorf("failed to batch save records: %w", err)
		}
	}
	return nil
}

func (db *DBStorage) Close() {
	db.pool.Close()
}

//go:embed db/migrations/*.sql
var migrationsDir embed.FS

func runMigrations(dsn string) error {
	d, err := iofs.New(migrationsDir, "db/migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return fmt.Errorf("failed to get a new migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
	}

	return nil
}
