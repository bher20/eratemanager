package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bher20/eratemanager/internal/alerting"
	"github.com/bher20/eratemanager/internal/metrics"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
	providerpkg "github.com/bher20/eratemanager/pkg/providers"
	"github.com/bher20/eratemanager/pkg/providers/electricproviders"
	"github.com/bher20/eratemanager/pkg/providers/waterproviders"
)

// buildRatesConfig creates a rates.Config with PDF paths from environment
// variables and provider defaults.
func buildRatesConfig() rates.Config {
	pdfPaths := make(map[string]string)
	for _, p := range electricproviders.GetAll() {
		// Check for env var override first (e.g., CEMC_PDF_PATH, NES_PDF_PATH)
		envKey := strings.ToUpper(p.Key()) + "_PDF_PATH"
		if path := os.Getenv(envKey); path != "" {
			pdfPaths[p.Key()] = path
		} else if p.DefaultPDFPath() != "" {
			pdfPaths[p.Key()] = p.DefaultPDFPath()
		}
	}
	return rates.Config{PDFPaths: pdfPaths}
}

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
	svc := rates.NewServiceWithStorage(buildRatesConfig(), st)

	// Determine which providers to process
	var providersToProcess []providerpkg.Provider
	var skippedResults []ProviderResult

	var allProviders []providerpkg.Provider
	for _, p := range electricproviders.GetAll() {
		allProviders = append(allProviders, p)
	}
	for _, p := range waterproviders.GetAll() {
		allProviders = append(allProviders, p)
	}

	jobName := "batch_refresh"
	started := time.Now()

	for _, p := range allProviders {
		// Check for fresh cache
		if cfg.CacheTTL > 0 {
			if isFresh, reason := isCacheFresh(ctx, st, p.Key(), cfg.CacheTTL); isFresh {
				log.Printf("batch: skipping %s (cache fresh: %s)", p.Key(), reason)
				skippedResults = append(skippedResults, ProviderResult{
					Provider:   p.Key(),
					Success:    true,
					Skipped:    true,
					SkipReason: reason,
				})
				continue
			}
		}

		// Check for resumed batch progress
		if cfg.ResumeFromProgress {
			progress, _ := st.GetBatchProgress(ctx, cfg.BatchID, p.Key())
			if progress != nil && progress.Status == "completed" {
				log.Printf("batch: skipping %s (already completed in this batch)", p.Key())
				skippedResults = append(skippedResults, ProviderResult{
					Provider:   p.Key(),
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
				Provider: p.Key(),
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
			results[i] = refreshProviderWithTracking(ctx, svc, st, p, cfg)

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
			go func(idx int, prov providerpkg.Provider) {
				defer wg.Done()
				sem <- struct{}{}        // acquire
				defer func() { <-sem }() // release

				results[idx] = refreshProviderWithTracking(ctx, svc, st, prov, cfg)
			}(i, p)
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
		runErr = fmt.Errorf("%d/%d providers failed", failCount, len(allProviders))
	}
	metrics.UpdateJobMetrics(jobName, started, runErr)
	dur := time.Since(started)

	log.Printf("batch: completed in %s — success: %d, failed: %d, skipped: %d",
		dur, successCount-skippedCount, failCount, skippedCount)

	// Send alert if there were failures
	if failCount > 0 {
		alert := alerting.BatchAlert{
			JobName:       jobName,
			TotalCount:    len(allProviders),
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
func refreshProviderWithTracking(ctx context.Context, svc *rates.Service, st storage.Storage, p providerpkg.Provider, cfg BatchConfig) ProviderResult {
	// Mark as in-progress
	_ = st.SaveBatchProgress(ctx, storage.BatchProgress{
		BatchID:   cfg.BatchID,
		Provider:  p.Key(),
		Status:    "in_progress",
		StartedAt: time.Now(),
	})

	result := refreshProviderWithRetry(ctx, svc, st, p, cfg)

	// Update progress based on result
	now := time.Now()
	progress := storage.BatchProgress{
		BatchID:     cfg.BatchID,
		Provider:    p.Key(),
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
func refreshProviderWithRetry(ctx context.Context, svc *rates.Service, st storage.Storage, p providerpkg.Provider, cfg BatchConfig) ProviderResult {
	result := ProviderResult{
		Provider: p.Key(),
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

			if p.Type() == providerpkg.ProviderTypeElectric {
				// Ensure PDF is present/refreshed
				if _, err := svc.RefreshPDF(attemptCtx, p.Key()); err != nil {
					return fmt.Errorf("refresh pdf: %w", err)
				}
				_, err := svc.GetElectricResidential(attemptCtx, p.Key())
				return err
			} else if p.Type() == providerpkg.ProviderTypeWater {
				waterSvc := rates.NewWaterServiceWithStorage(st)
				_, err := waterSvc.ForceRefresh(attemptCtx, p.Key())
				return err
			}
			return fmt.Errorf("unsupported provider type: %s", p.Type())
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
				p.Key(), attempt+1, cfg.RetryDelay, err)
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

	// Build rate fetching service
	svc := rates.NewServiceWithStorage(buildRatesConfig(), st)

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
			gotLock, err := st.AcquireAdvisoryLock(ctx, advisoryKey)
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
					if _, err := st.ReleaseAdvisoryLock(ctx, advisoryKey); err != nil {
						log.Printf("batch: lock release error: %v", err)
					}
				}()

				// ----- EXECUTE BATCH -----
				for _, p := range electricproviders.GetAll() {
					if _, err := svc.GetElectricResidential(ctx, p.Key()); err != nil {
						log.Printf("batch: provider %s refresh failed: %v", p.Key(), err)
						if runErr == nil {
							runErr = err
						}
					}
				}
				for _, p := range waterproviders.GetAll() {
					waterSvc := rates.NewWaterServiceWithStorage(st)
					if _, err := waterSvc.ForceRefresh(ctx, p.Key()); err != nil {
						log.Printf("batch: provider %s refresh failed: %v", p.Key(), err)
						if runErr == nil {
							runErr = err
						}
					}
				}
			}()

			// ----- METRICS -----
			metrics.UpdateJobMetrics(jobName, started, runErr)

			// (Optional enhancement: Save to scheduled_jobs through st.UpdateScheduledJob)
			dur := time.Since(started)
			errMsg := ""
			success := runErr == nil
			if runErr != nil {
				errMsg = runErr.Error()
			}
			if err := st.UpdateScheduledJob(ctx, jobName, started, dur, success, errMsg); err != nil {
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



// StartupRefreshConfig controls startup refresh behavior.
type StartupRefreshConfig struct {
	// Enabled controls whether startup refresh runs
	Enabled bool
	// CacheTTL is how long cached rates are considered fresh (skip refresh)
	CacheTTL time.Duration
	// ProviderTimeout is the max time for a single provider refresh
	ProviderTimeout time.Duration
	// NumWorkers is the number of concurrent worker goroutines
	NumWorkers int
	// UseLeaderElection enables advisory lock-based leader election (PostgreSQL only)
	UseLeaderElection bool
	// LeaderLockTimeout is how long to wait to acquire the leader lock
	LeaderLockTimeout time.Duration
}

// DefaultStartupRefreshConfig returns the default configuration for startup refresh.
func DefaultStartupRefreshConfig() StartupRefreshConfig {
	cfg := StartupRefreshConfig{
		Enabled:           true,
		CacheTTL:          24 * time.Hour,
		ProviderTimeout:   60 * time.Second,
		NumWorkers:        2,
		UseLeaderElection: true,
		LeaderLockTimeout: 5 * time.Second,
	}

	// Check environment variables for overrides
	if v := os.Getenv("STARTUP_REFRESH_ENABLED"); v != "" {
		cfg.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("STARTUP_REFRESH_CACHE_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.CacheTTL = time.Duration(n) * time.Hour
		}
	}
	if v := os.Getenv("STARTUP_REFRESH_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ProviderTimeout = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("STARTUP_REFRESH_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.NumWorkers = n
		}
	}
	if v := os.Getenv("STARTUP_REFRESH_LEADER_ELECTION"); v != "" {
		cfg.UseLeaderElection = v == "true" || v == "1"
	}

	return cfg
}

// AdvisoryLocker is an optional interface for storage backends that support advisory locks.
// PostgreSQL supports this for distributed leader election.
type AdvisoryLocker interface {
	AcquireAdvisoryLock(ctx context.Context, key int64) (bool, error)
	ReleaseAdvisoryLock(ctx context.Context, key int64) (bool, error)
}

// Advisory lock key for startup refresh leader election
const startupRefreshLockKey int64 = 13371338

// RunStartupRefresh runs a background refresh of missing or expired provider data.
// This is designed to be called asynchronously on server startup.
//
// Features:
// - Worker pool: Configurable number of concurrent workers process providers
// - Leader election: Only one replica runs refresh (PostgreSQL advisory locks)
// - Non-blocking: Never holds up application startup
// - Graceful: Handles context cancellation and timeouts
func RunStartupRefresh(ctx context.Context, st storage.Storage) {
	cfg := DefaultStartupRefreshConfig()

	if !cfg.Enabled {
		log.Println("startup-refresh: disabled via configuration")
		return
	}

	// Try leader election if enabled and storage supports it
	var locker AdvisoryLocker
	var hasLock bool

	if cfg.UseLeaderElection {
		if l, ok := st.(AdvisoryLocker); ok {
			locker = l

			// Try to acquire leader lock (non-blocking)
			lockCtx, cancel := context.WithTimeout(ctx, cfg.LeaderLockTimeout)
			acquired, err := locker.AcquireAdvisoryLock(lockCtx, startupRefreshLockKey)
			cancel()

			if err != nil {
				log.Printf("startup-refresh: failed to acquire leader lock: %v (proceeding anyway)", err)
				// Fall through - we'll run the refresh anyway since we can't coordinate
			} else if !acquired {
				log.Println("startup-refresh: another replica is handling refresh, skipping")
				return
			} else {
				hasLock = true
				log.Println("startup-refresh: acquired leader lock, this replica will handle refresh")
			}
		} else {
			log.Println("startup-refresh: leader election not available (storage doesn't support advisory locks)")
		}
	}

	// Ensure we release the lock when done
	defer func() {
		if hasLock && locker != nil {
			if _, err := locker.ReleaseAdvisoryLock(context.Background(), startupRefreshLockKey); err != nil {
				log.Printf("startup-refresh: failed to release leader lock: %v", err)
			} else {
				log.Println("startup-refresh: released leader lock")
			}
		}
	}()

	log.Printf("startup-refresh: starting with %d workers (cache_ttl=%s, timeout=%s)",
		cfg.NumWorkers, cfg.CacheTTL, cfg.ProviderTimeout)

	var allProviders []providerpkg.Provider
	for _, p := range electricproviders.GetAll() {
		allProviders = append(allProviders, p)
	}
	for _, p := range waterproviders.GetAll() {
		allProviders = append(allProviders, p)
	}

	svc := rates.NewServiceWithStorage(buildRatesConfig(), st)

	// Identify providers that need refresh
	var needsRefresh []providerpkg.Provider
	for _, p := range allProviders {
		snap, err := st.GetRatesSnapshot(ctx, p.Key())
		if err != nil || snap == nil || len(snap.Payload) == 0 {
			log.Printf("startup-refresh: %s needs refresh (no cached data)", p.Key())
			needsRefresh = append(needsRefresh, p)
			continue
		}

		age := time.Since(snap.FetchedAt)
		if age >= cfg.CacheTTL {
			log.Printf("startup-refresh: %s needs refresh (cache expired: %s ago, TTL: %s)", p.Key(), age.Round(time.Second), cfg.CacheTTL)
			needsRefresh = append(needsRefresh, p)
			continue
		}

		log.Printf("startup-refresh: %s cache is fresh (%s ago)", p.Key(), age.Round(time.Second))
	}

	if len(needsRefresh) == 0 {
		log.Println("startup-refresh: all providers have fresh cached data, nothing to refresh")
		return
	}

	log.Printf("startup-refresh: queuing %d providers for refresh: %v", len(needsRefresh), providerKeys(needsRefresh))

	// Create work queue
	workQueue := make(chan providerpkg.Provider, len(needsRefresh))
	for _, p := range needsRefresh {
		workQueue <- p
	}
	close(workQueue)

	// Track results
	type workerResult struct {
		provider string
		success  bool
		duration time.Duration
		err      error
	}
	results := make(chan workerResult, len(needsRefresh))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < cfg.NumWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for provider := range workQueue {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					results <- workerResult{provider: provider.Key(), err: ctx.Err()}
					continue
				default:
				}

				start := time.Now()
				providerCtx, cancel := context.WithTimeout(ctx, cfg.ProviderTimeout)

				// Refresh the provider
				err := refreshProvider(providerCtx, svc, st, provider)
				cancel()

				duration := time.Since(start)
				results <- workerResult{
					provider: provider.Key(),
					success:  err == nil,
					duration: duration,
					err:      err,
				}

				if err != nil {
					log.Printf("startup-refresh: worker-%d: %s failed after %s: %v",
						workerID, provider.Key, duration.Round(time.Millisecond), err)
				} else {
					log.Printf("startup-refresh: worker-%d: %s completed in %s",
						workerID, provider.Key, duration.Round(time.Millisecond))
				}
			}
		}(i)
	}

	// Wait for workers and collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	var successCount, failCount int
	for r := range results {
		if r.success {
			successCount++
		} else {
			failCount++
		}
	}

	log.Printf("startup-refresh: completed - success: %d, failed: %d", successCount, failCount)
}

// refreshProvider handles refreshing a single provider's data.
// refreshProvider refreshes a single provider (helper for startup refresh).
func refreshProvider(ctx context.Context, svc *rates.Service, st storage.Storage, p providerpkg.Provider) error {
	if p.Type() == providerpkg.ProviderTypeElectric {
		// Ensure PDF is present/refreshed
		if _, err := svc.RefreshPDF(ctx, p.Key()); err != nil {
			return fmt.Errorf("refresh pdf: %w", err)
		}
		_, err := svc.GetElectricResidential(ctx, p.Key())
		return err
	} else if p.Type() == providerpkg.ProviderTypeWater {
		waterSvc := rates.NewWaterServiceWithStorage(st)
		_, err := waterSvc.ForceRefresh(ctx, p.Key())
		return err
	}
	return fmt.Errorf("unsupported provider type: %s", p.Type())
}

// providerKeys extracts provider keys from a slice of Providers.
func providerKeys(provs []providerpkg.Provider) []string {
	keys := make([]string, len(provs))
	for i, p := range provs {
		keys[i] = p.Key()
	}
	return keys
}
