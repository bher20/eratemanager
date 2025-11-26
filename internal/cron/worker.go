package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bher20/eratemanager/internal/metrics"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
)

// buildRatesConfigWorker creates a rates.Config with PDF paths from environment
// variables and provider defaults (duplicate for worker package isolation).
func buildRatesConfigWorker() rates.Config {
	pdfPaths := make(map[string]string)
	for _, p := range rates.Providers() {
		envKey := strings.ToUpper(p.Key) + "_PDF_PATH"
		if path := os.Getenv(envKey); path != "" {
			pdfPaths[p.Key] = path
		} else if p.DefaultPDFPath != "" {
			pdfPaths[p.Key] = p.DefaultPDFPath
		}
	}
	return rates.Config{PDFPaths: pdfPaths}
}

// Run starts a simple cron worker that periodically refreshes provider rates
// using a Postgres pgxpool backend and PostgreSQL advisory locks so that in a
// multi-instance deployment only one worker executes the job.
func Run(ctx context.Context, driver, dsn string) error {
	if driver == "" {
		driver = "postgrespool"
	}
	if driver != "postgrespool" {
		return fmt.Errorf("cron worker requires ERATEMANAGER_DB_DRIVER=postgrespool (got %q)", driver)
	}

	// Open storage via the generic factory so that it still satisfies the
	// storage.Storage interface for rates.Service. We then assert the concrete
	// type to gain access to advisory locks.
	stGeneric, err := storage.Open(ctx, storage.Config{Driver: driver, DSN: dsn})
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer stGeneric.Close()

	pg, ok := stGeneric.(*storage.PostgresPoolStorage)
	if !ok {
		return fmt.Errorf("storage driver %q is not PostgresPoolStorage", driver)
	}

	// Build rates service with storage so results are cached to the DB.
	svc := rates.NewServiceWithStorage(buildRatesConfigWorker(), stGeneric)

	// Simple fixed-interval schedule; configurable via env.
	intervalSec := 300
	if raw := os.Getenv("ERATEMANAGER_CRON_INTERVAL_SECONDS"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			intervalSec = v
		}
	}
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	jobName := "refresh_rates"
	const lockKey int64 = 42

	log.Printf("cron worker starting, interval=%ds driver=%s", intervalSec, driver)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			started := time.Now()

			ok, err := pg.AcquireAdvisoryLock(ctx, lockKey)
			if err != nil {
				log.Printf("cron: acquire advisory lock failed: %v", err)
				metrics.UpdateJobMetrics(jobName, started, err)
				continue
			}
			if !ok {
				// Another worker is running this job.
				log.Printf("cron: advisory lock held by another worker, skipping run")
				continue
			}

			// We hold the lock for the duration of the job.
			var runErr error
			func() {
				defer func() {
					if _, err := pg.ReleaseAdvisoryLock(ctx, lockKey); err != nil {
						log.Printf("cron: release advisory lock failed: %v", err)
					}
				}()

				// Execute the job: refresh all known providers.
				for _, p := range rates.Providers() {
					if _, err := svc.GetResidential(ctx, p.Key); err != nil {
						log.Printf("cron: refresh provider %s failed: %v", p.Key, err)
						if runErr == nil {
							runErr = err
						}
					}
				}
			}()

			// Record metrics & job row.
			metrics.UpdateJobMetrics(jobName, started, runErr)
			dur := time.Since(started)
			errMsg := ""
			success := runErr == nil
			if runErr != nil {
				errMsg = runErr.Error()
			}
			if err := pg.UpdateScheduledJob(ctx, jobName, started, dur, success, errMsg); err != nil {
				log.Printf("cron: update scheduled_jobs failed: %v", err)
			}

			if runErr != nil {
				log.Printf("cron: job %s completed with error: %v (duration=%s)", jobName, runErr, dur)
			} else {
				log.Printf("cron: job %s completed successfully (duration=%s)", jobName, dur)
			}
		}
	}
}
