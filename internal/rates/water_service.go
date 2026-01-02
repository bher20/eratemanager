package rates

import (
	"context"
	"encoding/json"

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
func (s *WaterService) GetWaterRates(ctx context.Context, providerKey string) (*WaterRatesResponse, error) {
	parser, ok := GetWaterParser(providerKey)
	if !ok {
		return nil, nil // Provider not found
	}

	// Try cache first if we have storage
	if s.store != nil {
		snap, err := s.store.GetRatesSnapshot(ctx, "water:"+providerKey)
		if err == nil && snap != nil && len(snap.Payload) > 0 {
			var resp WaterRatesResponse
			if err := json.Unmarshal(snap.Payload, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	// Fetch from source
	provider, _ := GetProvider(providerKey)
	resp, err := parser.ParseHTML(provider.LandingURL)
	if err != nil {
		return nil, err
	}

	// Update cache if we have storage
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

// ForceRefresh bypasses the cache and fetches fresh rates.
func (s *WaterService) ForceRefresh(ctx context.Context, providerKey string) (*WaterRatesResponse, error) {
	parser, ok := GetWaterParser(providerKey)
	if !ok {
		return nil, nil
	}

	provider, _ := GetProvider(providerKey)
	resp, err := parser.ParseHTML(provider.LandingURL)
	if err != nil {
		return nil, err
	}

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
