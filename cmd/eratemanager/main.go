package main

import (
    "context"
    "log"
    "net/http"
    "os"

    "github.com/spf13/cobra"

    dbmigrate "github.com/bher20/eratemanager/internal/migrate"
    "github.com/bher20/eratemanager/internal/api"
    "github.com/bher20/eratemanager/internal/cron"
)

func main() {
    if err := rootCmd.Execute(); err != nil {
        log.Fatalf("error: %v", err)
    }
}

var rootCmd = &cobra.Command{
    Use:   "eratemanager",
    Short: "eRateManager server and tools",
    Long:  "eRateManager runs an HTTP API and provides database migration utilities.",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Default action: run the server.
        return serve()
    },
}

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Run the HTTP API server",
    RunE: func(cmd *cobra.Command, args []string) error {
        return serve()
    },
}

var migrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Database migrations (up, down, status)",
}

var migrateUpCmd = &cobra.Command{
    Use:   "up",
    Short: "Apply all up migrations",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := context.Background()
        driver, dsn := getDBEnv()
        log.Printf("running migrations up (driver=%s dsn=%s)", driver, dsn)
        return dbmigrate.Up(ctx, driver, dsn)
    },
}

var migrateDownCmd = &cobra.Command{
    Use:   "down",
    Short: "Rollback the most recent migration",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := context.Background()
        driver, dsn := getDBEnv()
        log.Printf("running migrations down (driver=%s dsn=%s)", driver, dsn)
        return dbmigrate.Down(ctx, driver, dsn)
    },
}

var migrateStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show migration status",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := context.Background()
        driver, dsn := getDBEnv()
        log.Printf("migration status (driver=%s dsn=%s)", driver, dsn)
        return dbmigrate.Status(ctx, driver, dsn)
    },
}


var cronCmd = &cobra.Command{
    Use:   "cron",
    Short: "Run background cron worker that refreshes rates on a schedule",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := context.Background()
        driver, dsn := getDBEnv()
        log.Printf("starting cron worker with driver=%s dsn=%s", driver, dsn)
        return cron.Run(ctx, driver, dsn)
    },
}

var batchCmd = &cobra.Command{
    Use:   "batch",
    Short: "Run a one-shot batch refresh of all provider rates (for CronJobs)",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := context.Background()
        driver, dsn := getDBEnv()
        log.Printf("starting batch refresh with driver=%s dsn=%s", driver, dsn)
        return cron.RunBatchOnce(ctx, driver, dsn)
    },
}

func init() {
    migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateStatusCmd)
    rootCmd.AddCommand(serveCmd, migrateCmd, cronCmd, batchCmd)
}

func serve() error {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }
    mux := api.NewMux()
    addr := ":" + port
    log.Printf("eRateManager listening on %s", addr)
    return http.ListenAndServe(addr, mux)
}

func getDBEnv() (driver, dsn string) {
    driver = os.Getenv("ERATEMANAGER_DB_DRIVER")
    dsn = os.Getenv("ERATEMANAGER_DB_DSN")
    if driver == "" {
        driver = "sqlite"
    }
    if dsn == "" {
        dsn = "eratemanager.db"
    }
    return
}
