package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"context"

	"github.com/bher20/eratemanager/internal/api/swagger"
	"github.com/bher20/eratemanager/internal/metrics"
	migrate "github.com/bher20/eratemanager/internal/migrate"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
	"github.com/bher20/eratemanager/internal/ui"
	"github.com/robfig/cron/v3"
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

	// Rates API - Electric (includes refresh endpoint)
	mux.HandleFunc("/rates/electric/", handleElectricRates(svc, st))

	RegisterProvidersHandler(mux)

	// Rates API - Water
	RegisterWaterHandlers(mux, st)

	// System Info
	mux.HandleFunc("/system/info", func(w http.ResponseWriter, r *http.Request) {
		drv := os.Getenv("ERATEMANAGER_DB_DRIVER")
		if drv == "" {
			drv = "sqlite"
		}

		// Format for display
		displayStorage := "SQLite"
		if drv == "postgres" {
			displayStorage = "PostgreSQL"
		} else if drv != "sqlite" {
			displayStorage = drv
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"storage": displayStorage,
		})
	})

	// Settings API
	mux.HandleFunc("/settings/refresh-interval", handleRefreshInterval(st))

	// Swagger API documentation
	mux.Handle("/swagger/", http.StripPrefix("/swagger", swagger.Handler()))

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

// handleElectricRates returns a handler that serves /rates/electric/{provider}/residential, /rates/electric/{provider}/pdf, and /rates/electric/{provider}/refresh.
func handleElectricRates(svc *rates.Service, st storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Expected paths: /rates/electric/{provider}/residential, /rates/electric/{provider}/pdf, or /rates/electric/{provider}/refresh
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) != 4 || parts[0] != "rates" || parts[1] != "electric" {
			metrics.RequestErrorsTotal.WithLabelValues("unknown", r.URL.Path, "404").Inc()
			http.NotFound(w, r)
			return
		}

		providerKey := strings.ToLower(parts[2])
		endpoint := parts[3]

		// Handle refresh
		if endpoint == "refresh" {
			handleElectricRefresh(w, r, providerKey, st, start)
			return
		}

		// Handle PDF download
		if endpoint == "pdf" {
			labelsPath := "/rates/electric/pdf"
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

// handleElectricRefresh handles the refresh endpoint for electric providers.
func handleElectricRefresh(w http.ResponseWriter, r *http.Request, providerKey string, st storage.Storage, start time.Time) {
	labelsPath := "/rates/electric/refresh"
	defer func() {
		dur := time.Since(start).Seconds()
		metrics.RequestDurationSeconds.WithLabelValues(providerKey, labelsPath).Observe(dur)
	}()

	metrics.RequestsTotal.WithLabelValues(providerKey).Inc()

	p, ok := rates.GetProvider(providerKey)
	if !ok {
		log.Printf("unknown provider %q for refresh", providerKey)
		metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
		http.NotFound(w, r)
		return
	}

	// Step 1: Download the latest PDF
	pdfURL, err := rates.RefreshProviderPDF(p)

	resp := RefreshResponse{
		Provider: providerKey,
		PDFURL:   pdfURL,
		Path:     p.DefaultPDFPath,
	}

	if err != nil {
		log.Printf("refresh PDF for %s failed: %v", providerKey, err)
		resp.Status = "error"
		resp.Error = err.Error()
		metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "500").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Step 2: Parse the PDF to extract rates
	ratesResp, parseErr := rates.ParseProviderPDF(providerKey)
	if parseErr != nil {
		log.Printf("parse PDF for %s failed: %v", providerKey, parseErr)
		resp.Status = "partial"
		resp.Error = "PDF downloaded but parsing failed: " + parseErr.Error()
	} else {
		resp.Status = "success"
		resp.Rates = ratesResp

		// Save to storage if available
		if st != nil && ratesResp != nil {
			payload, _ := json.Marshal(ratesResp)
			_ = st.SaveRatesSnapshot(r.Context(), storage.RatesSnapshot{
				Provider:  providerKey,
				Payload:   payload,
				FetchedAt: ratesResp.FetchedAt,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleRefreshInterval(st storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method == http.MethodGet {
			val, err := st.GetSetting(ctx, "refresh_interval_seconds")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if val == "" {
				val = "3600" // Default to 1 hour
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"interval": val})
			return
		}
		if r.Method == http.MethodPost {
			var req struct {
				Interval string `json:"interval"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Validate integer or cron expression
			if _, err := strconv.Atoi(req.Interval); err != nil {
				// Not an integer, check if it's a valid cron expression
				if _, cronErr := cron.ParseStandard(req.Interval); cronErr != nil {
					http.Error(w, "invalid interval or cron expression", http.StatusBadRequest)
					return
				}
			}
			if err := st.SetSetting(ctx, "refresh_interval_seconds", req.Interval); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
