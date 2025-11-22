package api

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"

    "github.com/bher20/eratemanager/internal/metrics"
    "github.com/bher20/eratemanager/internal/rates"
)

// NewMux constructs the HTTP mux, wiring in the rates service, metrics, and health endpoints.
func NewMux() *http.ServeMux {
    cfg := rates.Config{
        CEMCPDFPath: os.Getenv("CEMC_PDF_PATH"),
        NESPDFPath:  os.Getenv("NES_PDF_PATH"),
    }

    // Fallback to provider defaults if env vars are not set.
    if cfg.CEMCPDFPath == "" {
        if p, ok := rates.GetProvider("cemc"); ok && p.DefaultPDFPath != "" {
            cfg.CEMCPDFPath = p.DefaultPDFPath
        }
    }
    if cfg.NESPDFPath == "" {
        if p, ok := rates.GetProvider("nes"); ok && p.DefaultPDFPath != "" {
            cfg.NESPDFPath = p.DefaultPDFPath
        }
    }

    svc := rates.NewService(cfg)

    mux := http.NewServeMux()

    // Metrics endpoint.
    mux.Handle("/metrics", promhttp.Handler())

    // Health / readiness / liveness.
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })
    mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
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
    RegisterRefreshHandler(mux)

    return mux
}

// handleRates returns a handler that serves /rates/{provider}/residential using the rates.Service.
func handleRates(svc *rates.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Expected path: /rates/{provider}/residential
        path := strings.TrimPrefix(r.URL.Path, "/")
        parts := strings.Split(path, "/")
        if len(parts) != 3 || parts[0] != "rates" || parts[2] != "residential" {
            metrics.RequestErrorsTotal.WithLabelValues("unknown", r.URL.Path, "404").Inc()
            http.NotFound(w, r)
            return
        }

        providerKey := strings.ToLower(parts[1])
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
