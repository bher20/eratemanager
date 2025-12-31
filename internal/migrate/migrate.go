package migrate

import (
    "context"
    "database/sql"
    "embed"
    "fmt"

    "github.com/pressly/goose/v3"
    _ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func configureGoose() {
    goose.SetBaseFS(embedMigrations)
    goose.SetTableName("schema_migrations")
    goose.SetDialect("sqlite3")
}

func openDB(driver, dsn string) (*sql.DB, error) {
    if driver == "" {
        driver = "sqlite"
    }
    if dsn == "" {
        dsn = "eratemanager.db"
    }
    return sql.Open(driver, dsn)
}

func Up(ctx context.Context, driver, dsn string) error {
    configureGoose()
    db, err := openDB(driver, dsn)
    if err != nil { return err }
    defer db.Close()
    return goose.UpContext(ctx, db, "migrations")
}

func Down(ctx context.Context, driver, dsn string) error {
    configureGoose()
    db, err := openDB(driver, dsn)
    if err != nil { return err }
    defer db.Close()
    return goose.DownContext(ctx, db, "migrations")
}

func Status(ctx context.Context, driver, dsn string) error {
    configureGoose()
    db, err := openDB(driver, dsn)
    if err != nil { return err }
    defer db.Close()

    if err := goose.StatusContext(ctx, db, "migrations"); err != nil {
        return err
    }
    v, err := goose.GetDBVersion(db)
    if err != nil { return err }
    fmt.Printf("Current migration version: %d\n", v)
    return nil
}

func Version(ctx context.Context, driver, dsn string) (int64, error) {
    configureGoose()
    db, err := openDB(driver, dsn)
    if err != nil { return 0, err }
    defer db.Close()
    return goose.GetDBVersion(db)
}
