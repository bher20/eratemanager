package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bher20/eratemanager/internal/alerting"
	"github.com/bher20/eratemanager/internal/metrics"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
)

// BatchConfig controls batch processing behavior.
type BatchConfig struct {
	// MaxConcurrency limits parallel provider refreshes (0 = sequential)
	MaxConcurrency int
	// ProviderTimeout is the max time for a single provider refresh
	ProviderTimeout time.Duration
	// RetryAttempts is how many times to retry a failed provider
	RetryAttempts int
	// RetryDelay is the wait between retry attempts
	RetryDelay time.Duration
	// RateLimitDelay is the minimum time between starting provider refreshes
	RateLimitDelay time.Duration
	// CacheTTL is how long cached rates are considered fresh (skip re-parsing)
	CacheTTL time.Duration
	// ResumeFromProgress enables resuming incomplete batches
	ResumeFromProgress bool
	// BatchID identifies this batch run (for progress tracking)
	BatchID string
}

// DefaultBatchConfig returns sensible defaults for batch processing.
func DefaultBatchConfig() BatchConfig {
	cfg := BatchConfig{
		MaxConcurrency:     3,
		ProviderTimeout:    60 * time.Second,
		RetryAttempts:      2,
		RetryDelay:         5 * time.Second,
		RateLimitDelay:     500 * time.Millisecond,
		CacheTTL:           24 * time.Hour,
		ResumeFromProgress: true,
		BatchID:            fmt.Sprintf("batch_%d", time.Now().Unix()),
	}

	// Allow env overrides
	if v := os.Getenv("BATCH_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxConcurrency = n
		}
	}
	if v := os.Getenv("BATCH_PROVIDER_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ProviderTimeout = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("BATCH_RETRY_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.RetryAttempts = n
		}
	}
	if v := os.Getenv("BATCH_RATE_LIMIT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RateLimitDelay = time.Duration(n) * time.Millisecond
		}
	}
	if v := os.Getenv("BATCH_CACHE_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.CacheTTL = time.Duration(n) * time.Hour
		}
	}
	if v := os.Getenv("BATCH_RESUME_ENABLED"); v == "false" || v == "0" {
		cfg.ResumeFromProgress = false
	}
	if v := os.Getenv("BATCH_ID"); v != "" {
		cfg.BatchID = v
	}

	return cfg
}

// ProviderResult tracks the outcome of refreshing a single provider.
type ProviderResult struct {
	Provider   string
	Success    bool
	Duration   time.Duration
	Attempts   int
	Error      error
	Skipped    bool   // True if skipped due to fresh cache
	SkipReason string // Why it was skipped
}

// RunBatchOnce executes a single batch refresh of all provider rates.
// This is designed for Kubernetes CronJobs that run once and exit.
// It supports any storage driver (sqlite, postgres, postgrespool, memory).
//
// Features:
// - Database-first caching: Skip providers with fresh cached rates
// - Rate limiting: Configurable delay between provider refreshes
// - Alerting: Send webhook alerts on failures
// - Progress persistence: Track and resume incomplete batches
func RunBatchOnce(ctx context.Context, driver, dsn string) error {
	cfg := DefaultBatchConfig()
	log.Printf("batch: starting one-shot refresh with driver=%s concurrency=%d timeout=%s retries=%d cache_ttl=%s rate_limit=%s batch_id=%s",
		driver, cfg.MaxConcurrency, cfg.ProviderTimeout, cfg.RetryAttempts, cfg.CacheTTL, cfg.RateLimitDelay, cfg.BatchID)

	// Open storage
	st, err := storage.Open(ctx, storage.Config{Driver: driver, DSN: dsn})
	if err != nil {
		return fmt.Errorf("batch: open storage: %w", err)
	}
	defer st.Close()

	// Initialize alerter
	alertCfg := alerting.DefaultAlertConfig()
	alerter := alerting.NewAlerter(alertCfg)
	if alertCfg.Enabled {
		log.Printf("batch: alerting enabled (webhook type: %s)", alertCfg.WebhookType)
	}

	// Build rate fetching service
	svc := rates.NewServiceWithStorage(rates.Config{
		CEMCPDFPath: os.Getenv("CEMC_PDF_PATH"),
		NESPDFPath:  os.Getenv("NES_PDF_PATH"),
	}, st)

	providers := rates.Providers()
	jobName := "batch_refresh"
	started := time.Now()

	// Determine which providers to process
	var providersToProcess []rates.ProviderDescriptor
	var skippedResults []ProviderResult

	for _, p := range providers {
		// Check for fresh cache
		if cfg.CacheTTL > 0 {
			if isFresh, reason := isCacheFresh(ctx, st, p.Key, cfg.CacheTTL); isFresh {
				log.Printf("batch: skipping %s (cache fresh: %s)", p.Key, reason)
				skippedResults = append(skippedResults, ProviderResult{
					Provider:   p.Key,
					Success:    true,
					Skipped:    true,
					SkipReason: reason,
				})
				continue
			}
		}

		// Check for resumed batch progress
		if cfg.ResumeFromProgress {
			progress, _ := st.GetBatchProgress(ctx, cfg.BatchID, p.Key)
			if progress != nil && progress.Status == "completed" {
				log.Printf("batch: skipping %s (already completed in this batch)", p.Key)
				skippedResults = append(skippedResults, ProviderResult{
					Provider:   p.Key,
					Success:    true,
					Skipped:    true,
					SkipReason: "already completed in this batch",
				})
				continue
			}
		}

		providersToProcess = append(providersToProcess, p)

		// Initialize progress tracking
		if cfg.ResumeFromProgress {
			_ = st.SaveBatchProgress(ctx, storage.BatchProgress{
				BatchID:  cfg.BatchID,
				Provider: p.Key,
				Status:   "pending",
			})
		}
	}

	log.Printf("batch: processing %d providers (%d skipped)", len(providersToProcess), len(skippedResults))

	// Process providers
	results := make([]ProviderResult, len(providersToProcess))

	if cfg.MaxConcurrency <= 1 {
		// Sequential processing with rate limiting
		for i, p := range providersToProcess {
			results[i] = refreshProviderWithTracking(ctx, svc, st, p.Key, cfg)

			// Rate limiting between providers
			if i < len(providersToProcess)-1 && cfg.RateLimitDelay > 0 {
				select {
				case <-ctx.Done():
					break
				case <-time.After(cfg.RateLimitDelay):
				}
			}
		}
	} else {
		// Parallel processing with semaphore and rate limiting
		var wg sync.WaitGroup
		sem := make(chan struct{}, cfg.MaxConcurrency)
		rateLimiter := time.NewTicker(cfg.RateLimitDelay)
		defer rateLimiter.Stop()

		for i, p := range providersToProcess {
			// Rate limiting: wait for ticker before starting each goroutine
			if i > 0 && cfg.RateLimitDelay > 0 {
				select {
				case <-ctx.Done():
					break
				case <-rateLimiter.C:
				}
			}

			wg.Add(1)
			go func(idx int, providerKey string) {
				defer wg.Done()
				sem <- struct{}{}        // acquire
				defer func() { <-sem }() // release

				results[idx] = refreshProviderWithTracking(ctx, svc, st, providerKey, cfg)
			}(i, p.Key)
		}
		wg.Wait()
	}

	// Combine skipped and processed results
	allResults := append(skippedResults, results...)

	// Aggregate results
	var successCount, failCount, skippedCount int
	var failures []alerting.ProviderFailure
	for _, r := range allResults {
		if r.Skipped {
			skippedCount++
			successCount++ // Skipped counts as success
		} else if r.Success {
			successCount++
			log.Printf("batch: ✓ %s succeeded in %s (attempts: %d)", r.Provider, r.Duration, r.Attempts)
		} else {
			failCount++
			log.Printf("batch: ✗ %s failed after %d attempts: %v", r.Provider, r.Attempts, r.Error)
			failures = append(failures, alerting.ProviderFailure{
				Provider: r.Provider,
				Error:    r.Error.Error(),
				Attempts: r.Attempts,
			})
		}
	}

	// Record metrics
	var runErr error
	if failCount > 0 {
		runErr = fmt.Errorf("%d/%d providers failed", failCount, len(providers))
	}
	metrics.UpdateJobMetrics(jobName, started, runErr)
	dur := time.Since(started)

	log.Printf("batch: completed in %s — success: %d, failed: %d, skipped: %d",
		dur, successCount-skippedCount, failCount, skippedCount)

	// Send alert if there were failures
	if failCount > 0 {
		alert := alerting.BatchAlert{
			JobName:       jobName,
			TotalCount:    len(providers),
			SuccessCount:  successCount,
			FailedCount:   failCount,
			Duration:      dur,
			FailedDetails: failures,
			Timestamp:     started,
		}
		if err := alerter.SendBatchAlert(ctx, alert); err != nil {
			log.Printf("batch: failed to send alert: %v", err)
		}
	}

	if runErr != nil {
		return runErr
	}
	return nil
}

// isCacheFresh checks if the cached rates for a provider are still fresh.
func isCacheFresh(ctx context.Context, st storage.Storage, provider string, ttl time.Duration) (bool, string) {
	snap, err := st.GetRatesSnapshot(ctx, provider)
	if err != nil || snap == nil {
		return false, ""
	}

	age := time.Since(snap.FetchedAt)
	if age < ttl {
		return true, fmt.Sprintf("cached %s ago, TTL %s", age.Round(time.Second), ttl)
	}

	return false, ""
}

// refreshProviderWithTracking wraps refreshProviderWithRetry with progress tracking.
func refreshProviderWithTracking(ctx context.Context, svc *rates.Service, st storage.Storage, provider string, cfg BatchConfig) ProviderResult {
	// Mark as in-progress
	_ = st.SaveBatchProgress(ctx, storage.BatchProgress{
		BatchID:   cfg.BatchID,
		Provider:  provider,
		Status:    "in_progress",
		StartedAt: time.Now(),
	})

	result := refreshProviderWithRetry(ctx, svc, provider, cfg)

	// Update progress based on result
	now := time.Now()
	progress := storage.BatchProgress{
		BatchID:     cfg.BatchID,
		Provider:    provider,
		CompletedAt: now,
		RetryCount:  result.Attempts,
	}

	if result.Success {
		progress.Status = "completed"
	} else {
		progress.Status = "failed"
		if result.Error != nil {
			progress.ErrorMessage = result.Error.Error()
		}
	}

	_ = st.SaveBatchProgress(ctx, progress)

	return result
}

// refreshProviderWithRetry attempts to refresh a provider with retries.
func refreshProviderWithRetry(ctx context.Context, svc *rates.Service, provider string, cfg BatchConfig) ProviderResult {
	result := ProviderResult{
		Provider: provider,
		Attempts: 0,
	}

	started := time.Now()

	for attempt := 0; attempt <= cfg.RetryAttempts; attempt++ {
		result.Attempts = attempt + 1

		// Create timeout context for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, cfg.ProviderTimeout)

		err := func() error {
			defer cancel()

			// Check if parent context is done
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			_, err := svc.GetResidential(attemptCtx, provider)
			return err
		}()

		if err == nil {
			result.Success = true
			result.Duration = time.Since(started)
			return result
		}

		result.Error = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}

		// Wait before retry (unless last attempt)
		if attempt < cfg.RetryAttempts {
			log.Printf("batch: %s attempt %d failed, retrying in %s: %v",
				provider, attempt+1, cfg.RetryDelay, err)
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result
			case <-time.After(cfg.RetryDelay):
			}
		}
	}

	result.Duration = time.Since(started)
	return result
}

// RunBatch periodically refreshes ALL provider rates using *advisory locks*
// so that multiple replicas DO NOT run the batch simultaneously.
// This is designed for long-running deployments, not CronJobs.
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

// ForceRefreshProvider bypasses the cache and forces a fresh PDF parse for a provider.
// This is useful for manual refreshes triggered by the UI.
func ForceRefreshProvider(ctx context.Context, st storage.Storage, provider string) (*rates.RatesResponse, error) {
	svc := rates.NewServiceWithStorage(rates.Config{
		CEMCPDFPath: os.Getenv("CEMC_PDF_PATH"),
		NESPDFPath:  os.Getenv("NES_PDF_PATH"),
	}, st)

	// Force refresh by calling the internal method that always parses the PDF
	resp, err := svc.ForceRefresh(ctx, provider)
	if err != nil {
		return nil, err
	}

	// Save to storage
	if payload, err := json.Marshal(resp); err == nil {
		_ = st.SaveRatesSnapshot(ctx, storage.RatesSnapshot{
			Provider:  provider,
			Payload:   payload,
			FetchedAt: time.Now(),
		})
	}

	return resp, nil
}
