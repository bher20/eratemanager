package waterproviders

import "github.com/bher20/eratemanager/pkg/providers"

// WaterProvider is the interface that all water providers must implement.
type WaterProvider interface {
	providers.Provider

	// ParseHTML parses the rates from the provider's HTML page.
	ParseHTML(url string) (*WaterRatesResponse, error)
}
