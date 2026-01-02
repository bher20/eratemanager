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
	"github.com/robfig/cron/v3"
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

	// Initial interval from env or default
	// Can be integer seconds or cron expression
	intervalSetting := "300"
	if raw := os.Getenv("ERATEMANAGER_CRON_INTERVAL_SECONDS"); raw != "" {
		intervalSetting = raw
	}

	// Check DB for override
	if val, err := stGeneric.GetSetting(ctx, "refresh_interval_seconds"); err == nil && val != "" {
		intervalSetting = val
	}

	// Control loop ticker (check config and run time)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Helper to calculate next run time
	getNextRun := func(setting string, lastRun time.Time) time.Time {
		// Try integer seconds
		if v, err := strconv.Atoi(setting); err == nil && v > 0 {
			return lastRun.Add(time.Duration(v) * time.Second)
		}
		// Try cron expression
		if sched, err := cron.ParseStandard(setting); err == nil {
			return sched.Next(lastRun)
		}
		// Fallback to default 5m
		return lastRun.Add(5 * time.Minute)
	}

	// If starting fresh, run immediately, then schedule next
	nextRun := time.Now()

	jobName := "refresh_rates"
	const lockKey int64 = 42

	log.Printf("cron worker starting, initial setting=%q driver=%s", intervalSetting, driver)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 1. Check for config update
			if val, err := stGeneric.GetSetting(ctx, "refresh_interval_seconds"); err == nil && val != "" {
				if val != intervalSetting {
					log.Printf("cron: interval updated from %q to %q", intervalSetting, val)
					intervalSetting = val
					// Recalculate next run based on new setting and current time
					// If we were waiting for a long interval, we might want to run sooner if new schedule says so.
					// But simple approach: just calculate next run from NOW.
					nextRun = getNextRun(intervalSetting, time.Now())
				}
			}

			// 2. Check if it's time to run
			if time.Now().Before(nextRun) {
				continue
			}

			started := time.Now()

			ok, err := pg.AcquireAdvisoryLock(ctx, lockKey)
			if err != nil {
				log.Printf("cron: acquire advisory lock failed: %v", err)
				metrics.UpdateJobMetrics(jobName, started, err)
				// Retry soon? Or wait full interval?
				// If lock failed (DB error), maybe wait a bit.
				// If lock held (ok=false), wait full interval?
				// Original logic just continued loop, which waited for ticker.
				// Here we need to advance nextRun.
				nextRun = getNextRun(intervalSetting, time.Now())
				continue
			}
			if !ok {
				// Another worker is running this job.
				log.Printf("cron: advisory lock held by another worker, skipping run")
				nextRun = getNextRun(intervalSetting, time.Now())
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

			// Schedule next run
			nextRun = getNextRun(intervalSetting, time.Now())
		}
	}
}
