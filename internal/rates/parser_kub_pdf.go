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
// KUB is a TVA distributor - handles both their actual PDF format and test formats.
func ParseKubRatesFromText(text string) (*RatesResponse, error) {
	// === CUSTOMER CHARGE PATTERNS ===
	// KUB uses "Basic Service Charge: $20.50 per month"
	basicServiceRe := regexp.MustCompile(`Basic Service Charge[:\s]*\$([0-9]+(?:\.[0-9]+)?)\s*per month`)
	// Also try "Customer Charge" or "Service Charge" as fallback
	custRe := regexp.MustCompile(`(?:Customer|Service)\s+Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)
	// TVA Grid Access Charge - part of the monthly fixed charge
	gridAccessRe := regexp.MustCompile(`(?:TVA )?Grid Access Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)

	// === ENERGY CHARGE PATTERNS ===
	// KUB actual PDF format: "Summer Period    $0.11740 per kWh"
	summerRateRe := regexp.MustCompile(`Summer\s+Period\s+\$([0-9]+\.[0-9]+)\s*per kWh`)
	winterRateRe := regexp.MustCompile(`Winter\s+Period\s+\$([0-9]+\.[0-9]+)\s*per kWh`)
	transitionRateRe := regexp.MustCompile(`Transition\s+Period\s+\$([0-9]+\.[0-9]+)\s*per kWh`)

	// Test/alternate formats: "Energy Charge: 11.34 cents per kWh"
	energyCentsRe := regexp.MustCompile(`Energy Charge[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)
	// TVA format: "Energy Charge: Summer 9.5¢ per kWh"
	energyCentSymbolRe := regexp.MustCompile(`Energy Charge[:\s]*(?:Summer\s+)?([0-9]+(?:\.[0-9]+)?)\s*[¢c]\s*per kWh`)
	// Dollar format fallback
	energyUSDRe := regexp.MustCompile(`Energy Charge[:\s]*\$([0-9]+\.[0-9]+)\s*per kWh`)
	// Generic energy rate pattern: "$0.xxxxx per kWh"
	genericEnergyRe := regexp.MustCompile(`\$([0-9]+\.[0-9]{4,})\s*per kWh`)

	// === FUEL/ADJUSTMENT PATTERNS ===
	// KUB actual format: "Purchased Power Adjustment (1.053 cents per kWh)"
	ppaRe := regexp.MustCompile(`Purchased Power Adjustment\s*\(([0-9]+(?:\.[0-9]+)?)\s*cents? per kWh\)`)
	// Test format: "Fuel Cost Adjustment: 0.50 cents per kWh"
	fuelCentsRe := regexp.MustCompile(`(?:TVA )?Fuel(?: Cost)?\s*(?:Adjustment|Charge)[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)
	// Cent symbol format: "TVA Fuel Cost Adjustment: 0.25¢ per kWh"
	fuelCentSymbolRe := regexp.MustCompile(`(?:TVA )?Fuel(?: Cost)?\s*(?:Adjustment|Charge)[:\s]*([0-9]+(?:\.[0-9]+)?)\s*[¢c]\s*per kWh`)

	// === PARSE CUSTOMER CHARGE ===
	customerCharge := parseFirstFloat(basicServiceRe, text)
	if customerCharge == 0 {
		customerCharge = parseFirstFloat(custRe, text)
	}
	// Add Grid Access Charge if present (TVA format)
	if gridAccess := parseFirstFloat(gridAccessRe, text); gridAccess > 0 {
		customerCharge += gridAccess
	}

	// === PARSE ENERGY RATE ===
	energyRate := 0.0
	// Try KUB actual format first (dollars per kWh)
	if rate := parseFirstFloat(summerRateRe, text); rate > 0 {
		energyRate = rate
	} else if rate := parseFirstFloat(winterRateRe, text); rate > 0 {
		energyRate = rate
	} else if rate := parseFirstFloat(transitionRateRe, text); rate > 0 {
		energyRate = rate
	}
	// Try cents formats (test/alternate formats)
	if energyRate == 0 {
		if cents := parseFirstFloat(energyCentsRe, text); cents > 0 {
			energyRate = cents / 100.0
		} else if cents := parseFirstFloat(energyCentSymbolRe, text); cents > 0 {
			energyRate = cents / 100.0
		}
	}
	// Try dollar format fallbacks
	if energyRate == 0 {
		if rate := parseFirstFloat(energyUSDRe, text); rate > 0 {
			energyRate = rate
		} else if rate := parseFirstFloat(genericEnergyRe, text); rate > 0 {
			energyRate = rate
		}
	}

	// === PARSE FUEL RATE ===
	fuelRate := 0.0
	// Try Purchased Power Adjustment (KUB actual format, in cents)
	if ppaCents := parseFirstFloat(ppaRe, text); ppaCents > 0 {
		fuelRate = ppaCents / 100.0
	}
	// Try fuel adjustment formats (test formats, in cents)
	if fuelRate == 0 {
		if cents := parseFirstFloat(fuelCentsRe, text); cents > 0 {
			fuelRate = cents / 100.0
		} else if cents := parseFirstFloat(fuelCentSymbolRe, text); cents > 0 {
			fuelRate = cents / 100.0
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
				CustomerChargeMonthlyUSD: customerCharge,
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
