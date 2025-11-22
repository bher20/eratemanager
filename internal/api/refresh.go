package api

import (
    "encoding/json"
    "log"
    "net/http"
    "strings"
    "time"

    "github.com/bher20/eratemanager/internal/metrics"
    "github.com/bher20/eratemanager/internal/rates"
)

type RefreshResponse struct {
    Provider string `json:"provider"`
    PDFURL   string `json:"pdf_url"`
    Path     string `json:"path"`
    Status   string `json:"status"`
    Error    string `json:"error,omitempty"`
}

// RegisterRefreshHandler wires the /internal/refresh/{provider} endpoint into the mux.
func RegisterRefreshHandler(mux *http.ServeMux) {
    mux.HandleFunc("/internal/refresh/", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Path: /internal/refresh/{provider}
        path := strings.TrimPrefix(r.URL.Path, "/internal/refresh/")
        providerKey := strings.ToLower(strings.Trim(path, "/"))
        if providerKey == "" {
            http.NotFound(w, r)
            return
        }

        labelsProvider := providerKey
        labelsPath := "/internal/refresh"

        defer func() {
            dur := time.Since(start).Seconds()
            metrics.RequestDurationSeconds.WithLabelValues(labelsProvider, labelsPath).Observe(dur)
        }()

        metrics.RequestsTotal.WithLabelValues(labelsProvider).Inc()

        p, ok := rates.GetProvider(providerKey)
        if !ok {
            log.Printf("unknown provider %q for refresh", providerKey)
            metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "404").Inc()
            http.NotFound(w, r)
            return
        }

        pdfURL, err := rates.RefreshProviderPDF(p)

        resp := RefreshResponse{
            Provider: providerKey,
            PDFURL:   pdfURL,
            Path:     p.DefaultPDFPath,
        }

        if err != nil {
            log.Printf("refresh %s pdf failed: %v", providerKey, err)
            resp.Status = "error"
            resp.Error = err.Error()
            metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "500").Inc()
            w.WriteHeader(http.StatusInternalServerError)
        } else {
            resp.Status = "ok"
        }

        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(resp)
    })
}
