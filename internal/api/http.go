package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"context"

	"github.com/bher20/eratemanager/internal/metrics"
	migrate "github.com/bher20/eratemanager/internal/migrate"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
	"github.com/bher20/eratemanager/internal/ui"
)

// NewMux constructs the HTTP mux, wiring in the rates service, metrics, and health endpoints.
func NewMux() *http.ServeMux {
	// Build PDF paths map from environment variables and provider defaults
	pdfPaths := make(map[string]string)
	for _, p := range rates.Providers() {
		// Check for env var override first (e.g., CEMC_PDF_PATH, NES_PDF_PATH)
		envKey := strings.ToUpper(p.Key) + "_PDF_PATH"
		if path := os.Getenv(envKey); path != "" {
			pdfPaths[p.Key] = path
		} else if p.DefaultPDFPath != "" {
			pdfPaths[p.Key] = p.DefaultPDFPath
		}
	}
	cfg := rates.Config{PDFPaths: pdfPaths}

	// Optional auto-migration: run `goose up` on startup when enabled.
	autoMig := os.Getenv("ERATEMANAGER_AUTO_MIGRATE")
	driver := os.Getenv("ERATEMANAGER_DB_DRIVER")
	dsn := os.Getenv("ERATEMANAGER_DB_DSN")
	if autoMig == "1" || strings.ToLower(autoMig) == "true" || strings.ToLower(autoMig) == "yes" {
		ctx := context.Background()
		if driver == "" {
			driver = "sqlite"
		}
		if dsn == "" {
			dsn = "eratemanager.db"
		}
		if err := migrate.Up(ctx, driver, dsn); err != nil {
			log.Printf("auto-migration failed: %v", err)
		}
	}

	// Construct the rates service, preferring a real storage backend when available.
	var svc *rates.Service
	ctxSvc := context.Background()
	// When using the in-memory storage, preload provider descriptors so the
	// UI and cron workers know which providers exist without calling into
	// storage (avoids import cycles).
	var st storage.Storage
	var err error
	if driver == "memory" {
		// Convert rates.ProviderDescriptor -> storage.Provider
		var pList []storage.Provider
		for _, pd := range rates.Providers() {
			pList = append(pList, storage.Provider{
				Key:            pd.Key,
				Name:           pd.Name,
				LandingURL:     pd.LandingURL,
				DefaultPDFPath: pd.DefaultPDFPath,
				Notes:          pd.Notes,
			})
		}
		st = storage.NewMemoryWithProviders(pList)
		err = nil
	} else {
		st, err = storage.Open(ctxSvc, storage.Config{Driver: driver, DSN: dsn})
	}
	if err != nil {
		log.Printf("storage.Open failed (driver=%s dsn=%s): %v; falling back to PDF-only mode", driver, dsn, err)
		svc = rates.NewService(cfg)
	} else {
		log.Printf("rates service using storage backend driver=%s", driver)
		svc = rates.NewServiceWithStorage(cfg, st)
	}

	mux := http.NewServeMux()

	// Metrics endpoint.
	mux.Handle("/metrics", promhttp.Handler())

	// Health / readiness / liveness.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		drv := os.Getenv("ERATEMANAGER_DB_DRIVER")
		dsn := os.Getenv("ERATEMANAGER_DB_DSN")
		if drv == "" {
			drv = "sqlite"
		}
		if dsn == "" {
			dsn = "eratemanager.db"
		}
		st, err := storage.Open(ctx, storage.Config{Driver: drv, DSN: dsn})
		if err != nil {
			log.Printf("readyz: storage open failed: %v", err)
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		defer st.Close()
		if err := st.Ping(ctx); err != nil {
			log.Printf("readyz: db ping failed: %v", err)
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live"))
	})

	// Rates API.
	mux.HandleFunc("/rates/", handleRates(svc))

	// Internal refresh endpoint for CronJobs / manual refresh.
	RegisterRefreshHandler(mux, st)
	RegisterProvidersHandler(mux)

	// Web UI
	mux.Handle("/ui/", http.StripPrefix("/ui/", ui.Handler()))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	return mux
}

// handleRates returns a handler that serves /rates/{provider}/residential and /rates/{provider}/pdf using the rates.Service.
func handleRates(svc *rates.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Expected paths: /rates/{provider}/residential or /rates/{provider}/pdf
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) != 3 || parts[0] != "rates" {
			metrics.RequestErrorsTotal.WithLabelValues("unknown", r.URL.Path, "404").Inc()
			http.NotFound(w, r)
			return
		}

		providerKey := strings.ToLower(parts[1])
		endpoint := parts[2]

		// Handle PDF download
		if endpoint == "pdf" {
			labelsPath := "/rates/pdf"
			defer func() {
				dur := time.Since(start).Seconds()
				metrics.RequestDurationSeconds.WithLabelValues(providerKey, labelsPath).Observe(dur)
			}()
			metrics.RequestsTotal.WithLabelValues(providerKey).Inc()

			p, ok := rates.GetProvider(providerKey)
			if !ok {
				metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
				http.NotFound(w, r)
				return
			}

			pdfPath := p.DefaultPDFPath
			if pdfPath == "" {
				metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
				http.Error(w, "no PDF configured for this provider", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_rates.pdf", providerKey))
			http.ServeFile(w, r, pdfPath)
			return
		}

		// Handle residential rates
		if endpoint != "residential" {
			metrics.RequestErrorsTotal.WithLabelValues("unknown", r.URL.Path, "404").Inc()
			http.NotFound(w, r)
			return
		}

		labelsProvider := providerKey
		labelsPath := "/rates/residential"

		defer func() {
			dur := time.Since(start).Seconds()
			metrics.RequestDurationSeconds.WithLabelValues(labelsProvider, labelsPath).Observe(dur)
		}()

		metrics.RequestsTotal.WithLabelValues(labelsProvider).Inc()

		resp, err := svc.GetResidential(r.Context(), providerKey)
		if err != nil {
			log.Printf("get residential rates for %s failed: %v", providerKey, err)
			metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "500").Inc()
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("encode response failed: %v", err)
			metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "500").Inc()
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}
