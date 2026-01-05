package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations
var embedMigrations embed.FS

func configureGoose(driver string) error {
	goose.SetBaseFS(embedMigrations)
	goose.SetTableName("schema_migrations")

	if driver == "sqlite" || driver == "sqlite3" {
		return goose.SetDialect("sqlite3")
	}
	if driver == "postgres" || driver == "pgx" || driver == "postgrespool" {
		return goose.SetDialect("postgres")
	}
	return fmt.Errorf("unsupported driver for goose: %s", driver)
}

func getMigrationDir(driver string) string {
	if driver == "postgres" || driver == "pgx" || driver == "postgrespool" {
		return "migrations/postgres"
	}
	return "migrations/sqlite"
}

func openDB(driver, dsn string) (*sql.DB, error) {
	if driver == "" {
		driver = "sqlite"
	}
	if dsn == "" {
		dsn = "eratemanager.db"
	}

	// Map custom driver names to stdlib drivers
	if driver == "postgrespool" {
		driver = "pgx"
	}
	if driver == "postgres" {
		driver = "pgx"
	}

	return sql.Open(driver, dsn)
}

func Up(ctx context.Context, driver, dsn string) error {
	if err := configureGoose(driver); err != nil {
		return err
	}
	db, err := openDB(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return goose.UpContext(ctx, db, getMigrationDir(driver))
}

func Down(ctx context.Context, driver, dsn string) error {
	if err := configureGoose(driver); err != nil {
		return err
	}
	db, err := openDB(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return goose.DownContext(ctx, db, getMigrationDir(driver))
}

func Status(ctx context.Context, driver, dsn string) error {
	if err := configureGoose(driver); err != nil {
		return err
	}
	db, err := openDB(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return goose.Status(db, getMigrationDir(driver))
}
