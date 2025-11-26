package rates

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"time"

	pdf "github.com/ledongthuc/pdf"
)

func init() {
	RegisterParser(ParserConfig{
		Key:       "kub",
		Name:      "Knoxville Utilities Board",
		ParsePDF:  ParseKubRatesFromPDF,
		ParseText: ParseKubRatesFromText,
	})
}

// ParseKubRatesFromPDF opens a Knoxville Utilities Board rates PDF at the given path,
// extracts text, and delegates to ParseKubRatesFromText.
func ParseKubRatesFromPDF(path string) (*RatesResponse, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	rc, err := r.GetPlainText()
	if err != nil {
		return nil, fmt.Errorf("extract pdf text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rc); err != nil {
		return nil, fmt.Errorf("read pdf text: %w", err)
	}

	return ParseKubRatesFromText(buf.String())
}

// ParseKubRatesFromText parses a plain-text representation of the
// Knoxville Utilities Board residential rates and extracts fields using regex heuristics.
// KUB is a TVA distributor similar to NES, so the rate structure is similar.
func ParseKubRatesFromText(text string) (*RatesResponse, error) {
	// KUB may use "Customer Charge", "Service Charge", or "Basic Service Charge"
	custRe := regexp.MustCompile(`(?:Customer|Service|Basic Service)\s+Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)

	// TVA Grid Access Charge - part of the monthly fixed charge
	gridAccessRe := regexp.MustCompile(`(?:TVA )?Grid Access Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)

	// Energy Charge patterns - try cents first (more common for TVA distributors)
	// Format: "Energy Charge: 9.254¢ per kWh" or "Energy Charge: Summer 9.254 cents per kWh"
	energyCentsRe := regexp.MustCompile(`Energy Charge[:\s]*(?:Summer\s+(?:Period\s+)?)?([0-9]+(?:\.[0-9]+)?)\s*[¢c]`)
	energyCentsAltRe := regexp.MustCompile(`Energy Charge[:\s]*(?:Summer\s+)?([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)
	energyUSDRe := regexp.MustCompile(`Energy Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*per kWh`)

	// Fuel Cost Adjustment (TVA)
	fuelRe := regexp.MustCompile(`(?:TVA )?Fuel(?: Cost)?\s*(?:Adjustment|Charge)[:\s]*([0-9]+(?:\.[0-9]+)?)\s*[¢c]?(?:ents?)?\s*(?:per kWh)?`)

	// Parse customer charge components
	customerCharge := parseFirstFloat(custRe, text)
	gridAccessCharge := parseFirstFloat(gridAccessRe, text)

	// Total customer charge = service charge + grid access charge
	totalCustomerCharge := customerCharge
	if gridAccessCharge > 0 {
		totalCustomerCharge += gridAccessCharge
	}

	// Parse energy rate - try cents format first (common for TVA)
	energyRate := 0.0
	if cents := parseFirstFloat(energyCentsRe, text); cents > 0 {
		energyRate = cents / 100.0
	} else if cents := parseFirstFloat(energyCentsAltRe, text); cents > 0 {
		energyRate = cents / 100.0
	} else if usd := parseFirstFloat(energyUSDRe, text); usd > 0 {
		energyRate = usd
	}

	// Parse fuel rate
	fuelRate := 0.0
	if v := parseFirstFloat(fuelRe, text); v > 0 {
		// If value looks like it's in cents (> 1), convert
		if v > 1 {
			fuelRate = v / 100.0
		} else {
			fuelRate = v
		}
	}

	energyCents := energyRate * 100
	fuelCents := fuelRate * 100

	now := time.Now().UTC()
	rawCopy := text

	resp := &RatesResponse{
		Utility:   "KUB",
		Source:    "KUB Residential Rates PDF",
		SourceURL: "https://www.kub.org/bills-payments/understand-your-bill/residential-rates/",
		FetchedAt: now,
		Rates: Rates{
			ResidentialStandard: ResidentialStandard{
				IsPresent:                true,
				CustomerChargeMonthlyUSD: totalCustomerCharge,
				EnergyRateUSDPerKWh:      energyRate,
				EnergyRateCentsPerKWh:    energyCents,
				TVAFuelRateUSDPerKWh:     fuelRate,
				TVAFuelRateCentsPerKWh:   fuelCents,
				RawSection:               &rawCopy,
			},
		},
	}
	return resp, nil
}
