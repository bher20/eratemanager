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
    custRe := regexp.MustCompile(`Customer Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)`)
    // Some utilities express energy charge directly in $/kWh.
    energyUSDRe := regexp.MustCompile(`Energy Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*per kWh`)
    // Others use cents per kWh.
    energyCentsRe := regexp.MustCompile(`Energy Charge[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)
    fuelRe := regexp.MustCompile(`Fuel(?: Cost)? Adjustment[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)

    customerCharge := parseFirstFloat(custRe, text)

    energyRate := parseFirstFloat(energyUSDRe, text)
    if energyRate == 0 {
        cents := parseFirstFloat(energyCentsRe, text)
        energyRate = cents / 100.0
    }

    fuelRate := 0.0
    if v := parseFirstFloat(fuelRe, text); v > 0 {
        fuelRate = v / 100.0
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
