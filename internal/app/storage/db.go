package storage

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

// PostgreSQL storage
type DBStorage struct {
	pool *pgxpool.Pool
}

// New PostgreSQL storage
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

// Find record by original URL
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

// Find record by shortened path
func (db *DBStorage) FindByShortenedPath(ctx context.Context, shortenedPath string) (models.Record, error) {
	row := db.pool.QueryRow(
		ctx,
		`SELECT "original_url", "correlation_id", "is_deleted"
		 FROM "urls" WHERE "shortened_path" = @shortenedPath`,
		pgx.NamedArgs{"shortenedPath": shortenedPath},
	)
	var originalURL, correlationID string
	var isDeleted bool
	err := row.Scan(&originalURL, &correlationID, &isDeleted)
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
		IsDeleted:     isDeleted,
	}, nil
}

// Find user records
func (db *DBStorage) FindByUser(ctx context.Context, user models.User) ([]models.Record, error) {
	rows, err := db.pool.Query(
		ctx,
		`SELECT "original_url", "shortened_path", "correlation_id", "user_id"
		 FROM "urls"
		 WHERE "user_id" = @userID`,
		pgx.NamedArgs{"userID": user.ID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records: %s", err.Error())
	}

	result, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Record, error) {
		var originalURL, shortenedPath, correlationID string
		var userID int
		err = row.Scan(&originalURL, &shortenedPath, &correlationID, &userID)

		return models.Record{
			OriginalURL:   originalURL,
			ShortenedPath: shortenedPath,
			CorrelationID: correlationID,
			UserID:        userID,
		}, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records: %s", err.Error())
	}

	return result, nil
}

// Save record to database
func (db *DBStorage) Save(ctx context.Context, record models.Record) error {
	_, err := db.pool.Exec(
		ctx,
		`INSERT INTO "urls" ("original_url", "shortened_path", "user_id") VALUES (@originalURL, @shortenedPath, @user_id)`,
		pgx.NamedArgs{
			"originalURL":   record.OriginalURL,
			"shortenedPath": record.ShortenedPath,
			"user_id":       record.UserID,
		},
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

// Batch save records to database
func (db *DBStorage) BatchSave(ctx context.Context, records []models.Record) error {
	batch := &pgx.Batch{}
	for _, r := range records {
		batch.Queue(
			`INSERT INTO "urls" ("original_url", "shortened_path", "user_id") VALUES ($1, $2, $3)
			 ON CONFLICT ("original_url") DO UPDATE SET "shortened_path" = $2`,
			r.OriginalURL, r.ShortenedPath, r.UserID,
		)
	}
	res := db.pool.SendBatch(ctx, batch)
	defer res.Close()

	for range records {
		_, err := res.Exec()
		if err != nil {
			return fmt.Errorf("failed to batch save: %w", err)
		}
	}

	return res.Close()
}

// Batch delete records from database
func (db *DBStorage) BatchDelete(ctx context.Context, records []models.Record) error {
	batch := pgx.Batch{}
	for _, r := range records {
		batch.Queue(
			`UPDATE "urls" SET "is_deleted" = TRUE
			 WHERE "shortened_path" = @shortenedPath AND "user_id" = @userID`,
			pgx.NamedArgs{"shortenedPath": r.ShortenedPath, "userID": r.UserID},
		)
	}
	err := db.pool.SendBatch(ctx, &batch).Close()
	if err != nil {
		return fmt.Errorf("failed to batch delete: %s", err.Error())
	}

	return nil
}

// Create user
func (db *DBStorage) CreateUser(ctx context.Context) (models.User, error) {
	row := db.pool.QueryRow(ctx, `INSERT INTO "users" ("id") VALUES (DEFAULT) RETURNING "id"`)
	user := models.User{}
	err := row.Scan(&user.ID)
	if err != nil {
		return user, fmt.Errorf("failed to create user: %s", err.Error())
	}

	return user, nil
}

// Close connection to database
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
