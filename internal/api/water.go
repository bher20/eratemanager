package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bher20/eratemanager/internal/metrics"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
)

// WaterService coordinates fetching and caching of water rates.
type WaterService struct {
	store storage.Storage // may be nil for direct fetch mode
}

// NewWaterService creates a water service without storage.
func NewWaterService() *WaterService {
	return &WaterService{}
}

// NewWaterServiceWithStorage creates a water service with storage backend.
func NewWaterServiceWithStorage(st storage.Storage) *WaterService {
	return &WaterService{store: st}
}

// GetWaterRates returns water rates for a provider.
func (s *WaterService) GetWaterRates(ctx context.Context, providerKey string) (*rates.WaterRatesResponse, error) {
	parser, ok := rates.GetWaterParser(providerKey)
	if !ok {
		return nil, nil // Provider not found
	}

	// Try cache first if we have storage
	if s.store != nil {
		snap, err := s.store.GetRatesSnapshot(ctx, "water:"+providerKey)
		if err == nil && snap != nil && len(snap.Payload) > 0 {
			var resp rates.WaterRatesResponse
			if err := json.Unmarshal(snap.Payload, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	// Fetch from source
	provider, _ := rates.GetProvider(providerKey)
	resp, err := parser.ParseHTML(provider.LandingURL)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.store != nil && resp != nil {
		if payload, err := json.Marshal(resp); err == nil {
			_ = s.store.SaveRatesSnapshot(ctx, storage.RatesSnapshot{
				Provider:  "water:" + providerKey,
				Payload:   payload,
				FetchedAt: resp.FetchedAt,
			})
		}
	}

	return resp, nil
}

// RegisterWaterHandlers registers all water-related HTTP handlers.
func RegisterWaterHandlers(mux *http.ServeMux, st storage.Storage) {
	var waterSvc *WaterService
	if st != nil {
		waterSvc = NewWaterServiceWithStorage(st)
	} else {
		waterSvc = NewWaterService()
	}

	// Water providers list
	mux.HandleFunc("/water/providers", handleWaterProviders)

	// Water rates endpoint
	mux.HandleFunc("/water/rates/", handleWaterRates(waterSvc))

	// Water refresh endpoint
	mux.HandleFunc("/internal/refresh/water/", handleWaterRefresh(waterSvc))
}

// handleWaterProviders returns the list of water providers.
func handleWaterProviders(w http.ResponseWriter, r *http.Request) {
	providers := rates.WaterProviders()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(providers); err != nil {
		log.Printf("encode water providers failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

// handleWaterRates returns a handler for /water/rates/{provider}
func handleWaterRates(svc *WaterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Expected path: /water/rates/{provider}
		path := strings.TrimPrefix(r.URL.Path, "/water/rates/")
		providerKey := strings.ToLower(strings.TrimSuffix(path, "/"))

		if providerKey == "" {
			http.NotFound(w, r)
			return
		}

		labelsPath := "/water/rates"
		defer func() {
			dur := time.Since(start).Seconds()
			metrics.RequestDurationSeconds.WithLabelValues(providerKey, labelsPath).Observe(dur)
		}()

		metrics.RequestsTotal.WithLabelValues(providerKey).Inc()

		resp, err := svc.GetWaterRates(r.Context(), providerKey)
		if err != nil {
			log.Printf("get water rates for %s failed: %v", providerKey, err)
			metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "500").Inc()
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if resp == nil {
			metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("encode water response failed: %v", err)
			metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "500").Inc()
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

// handleWaterRefresh handles /internal/refresh/water/{provider}
func handleWaterRefresh(svc *WaterService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/internal/refresh/water/")
		providerKey := strings.ToLower(strings.TrimSuffix(path, "/"))

		if providerKey == "" {
			http.Error(w, "provider key required", http.StatusBadRequest)
			return
		}

		// Force refresh by fetching directly from source
		parser, ok := rates.GetWaterParser(providerKey)
		if !ok {
			http.Error(w, "unknown water provider", http.StatusNotFound)
			return
		}

		provider, _ := rates.GetProvider(providerKey)
		resp, err := parser.ParseHTML(provider.LandingURL)
		if err != nil {
			log.Printf("refresh water rates for %s failed: %v", providerKey, err)
			http.Error(w, "refresh failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update cache if we have storage
		if svc.store != nil && resp != nil {
			if payload, err := json.Marshal(resp); err == nil {
				_ = svc.store.SaveRatesSnapshot(r.Context(), storage.RatesSnapshot{
					Provider:  "water:" + providerKey,
					Payload:   payload,
					FetchedAt: resp.FetchedAt,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "refreshed",
			"provider": providerKey,
			"rates":    resp,
		})
	}
}
