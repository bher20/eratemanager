package rates

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"time"

	pdf "github.com/ledongthuc/pdf"
)

// ParseNESRatesFromPDF opens a NES rates PDF at the given path, extracts
// text, and delegates to ParseNESRatesFromText.
func ParseNESRatesFromPDF(path string) (*RatesResponse, error) {
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

	return ParseNESRatesFromText(buf.String())
}

// ParseNESRatesFromText parses a plain-text representation of the NES
// residential rates and extracts fields using regex heuristics.
func ParseNESRatesFromText(text string) (*RatesResponse, error) {
	// NES uses "Service Charge" instead of "Customer Charge"
	// Format: "Service Charge: $14.06 per month" or similar
	custRe := regexp.MustCompile(`(?:Customer|Service)\s+Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)

	// NES Energy Charge format: "Energy Charge: Summer Period 9.254¢ per kWh"
	// or "Energy Charge: 9.254¢ per kWh per month"
	energyCentsRe := regexp.MustCompile(`Energy Charge[:\s]*(?:Summer Period\s+)?([0-9]+(?:\.[0-9]+)?)\s*[¢c]`)

	// Fallback: Some utilities express energy charge directly in $/kWh
	energyUSDRe := regexp.MustCompile(`Energy Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*per kWh`)

	// Also try cents per kWh format
	energyCentsAltRe := regexp.MustCompile(`Energy Charge[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)

	// Fuel adjustment (TVA)
	fuelRe := regexp.MustCompile(`Fuel(?: Cost)? Adjustment[:\s]*([0-9]+(?:\.[0-9]+)?)\s*[¢c]?(?:ents?)?\s*per kWh`)

	// TVA Grid Access Charge - this is part of the monthly charge
	gridAccessRe := regexp.MustCompile(`(?:TVA )?Grid Access Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*per month`)

	customerCharge := parseFirstFloat(custRe, text)
	gridAccessCharge := parseFirstFloat(gridAccessRe, text)

	// Add grid access charge to customer charge if found
	totalCustomerCharge := customerCharge
	if gridAccessCharge > 0 {
		totalCustomerCharge += gridAccessCharge
	}

	// Try to extract energy rate - prefer cents format for NES
	energyRate := 0.0
	if cents := parseFirstFloat(energyCentsRe, text); cents > 0 {
		energyRate = cents / 100.0
	} else if usd := parseFirstFloat(energyUSDRe, text); usd > 0 {
		energyRate = usd
	} else if cents := parseFirstFloat(energyCentsAltRe, text); cents > 0 {
		energyRate = cents / 100.0
	}

	fuelRate := 0.0
	if v := parseFirstFloat(fuelRe, text); v > 0 {
		// If value looks like cents (small number), convert
		if v < 1 {
			fuelRate = v // Already in dollars
		} else {
			fuelRate = v / 100.0
		}
	}

	energyCents := energyRate * 100
	fuelCents := fuelRate * 100

	now := time.Now().UTC()
	rawCopy := text

	resp := &RatesResponse{
		Utility:   "NES",
		Source:    "NES Residential Rates PDF",
		SourceURL: "https://www.nespower.com/rates/",
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
