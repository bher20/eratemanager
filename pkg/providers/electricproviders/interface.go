package electricproviders

import "github.com/bher20/eratemanager/pkg/providers"

// ElectricProvider is the interface that all electric providers must implement.
type ElectricProvider interface {
	providers.Provider

	// ParsePDF parses the rates from a PDF file at the given path.
	ParsePDF(path string) (*ElectricRatesResponse, error)

	// ParseText parses the rates from extracted text (useful for testing).
	ParseText(text string) (*ElectricRatesResponse, error)
}
