package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bher20/eratemanager/internal/metrics"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
)

// RunBatch periodically refreshes ALL provider rates using *advisory locks*
// so that multiple replicas DO NOT run the batch simultaneously.
func RunBatch(ctx context.Context, driver, dsn string) error {
	if driver != "postgrespool" {
		return fmt.Errorf("batch worker requires ERATEMANAGER_DB_DRIVER=postgrespool (got %q)", driver)
	}

	// open DB
	st, err := storage.Open(ctx, storage.Config{Driver: driver, DSN: dsn})
	if err != nil {
		return fmt.Errorf("batch: open storage: %w", err)
	}
	defer st.Close()

	pg, ok := st.(*storage.PostgresPoolStorage)
	if !ok {
		return fmt.Errorf("batch: storage is not PostgresPoolStorage")
	}

	// Build rate fetching service
	svc := rates.NewServiceWithStorage(rates.Config{
		CEMCPDFPath: os.Getenv("CEMC_PDF_PATH"),
		NESPDFPath:  os.Getenv("NES_PDF_PATH"),
	}, st)

	// Configurable interval
	intervalSec := 3600
	if raw := os.Getenv("BATCH_INTERVAL_SECONDS"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			intervalSec = v
		}
	}

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	jobName := "batch_refresh"
	const advisoryKey int64 = 13371337 // unique lock key

	log.Printf("batch worker starting: interval=%ds driver=postgrespool", intervalSec)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			started := time.Now()

			// ----- ACQUIRE LOCK -----
			gotLock, err := pg.AcquireAdvisoryLock(ctx, advisoryKey)
			if err != nil {
				log.Printf("batch: lock acquire error: %v", err)
				metrics.UpdateJobMetrics(jobName, started, err)
				continue
			}
			if !gotLock {
				log.Printf("batch: lock held by another node — skipping this cycle")
				continue
			}

			var runErr error
			func() {
				defer func() {
					// always release
					if _, err := pg.ReleaseAdvisoryLock(ctx, advisoryKey); err != nil {
						log.Printf("batch: lock release error: %v", err)
					}
				}()

				// ----- EXECUTE BATCH -----
				for _, p := range rates.Providers() {
					if _, err := svc.GetResidential(ctx, p.Key); err != nil {
						log.Printf("batch: provider %s refresh failed: %v", p.Key, err)
						if runErr == nil {
							runErr = err
						}
					}
				}
			}()

			// ----- METRICS -----
			metrics.UpdateJobMetrics(jobName, started, runErr)

			// (Optional enhancement: Save to scheduled_jobs through pg.UpdateScheduledJob)
			dur := time.Since(started)
			errMsg := ""
			success := runErr == nil
			if runErr != nil {
				errMsg = runErr.Error()
			}
			if err := pg.UpdateScheduledJob(ctx, jobName, started, dur, success, errMsg); err != nil {
				log.Printf("batch: update scheduled_jobs failed: %v", err)
			}

			if runErr != nil {
				log.Printf("batch: run completed WITH ERROR: %v", runErr)
			} else {
				log.Printf("batch: run completed successfully")
			}
		}
	}
}
