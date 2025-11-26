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
		Key:       "cemc",
		Name:      "Cumberland Electric Membership Corporation",
		ParsePDF:  ParseCEMCRatesFromPDF,
		ParseText: ParseCEMCRatesFromText,
	})
}

// ParseCEMCRatesFromPDF opens a CEMC rates PDF at the given path, extracts
// text, and delegates to ParseCEMCRatesFromText.
func ParseCEMCRatesFromPDF(path string) (*RatesResponse, error) {
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

	return ParseCEMCRatesFromText(buf.String())
}

// ParseCEMCRatesFromText parses a plain-text representation of the CEMC
// rates PDF and extracts the residential standard fields using regex.
func ParseCEMCRatesFromText(text string) (*RatesResponse, error) {
	// Try to narrow to the residential RS section.
	rsRe := regexp.MustCompile(`RESIDENTIAL RATE[^\n]*SCHEDULE RS(?s)(.+?)(?:SUPPLEMENTAL RESIDENTIAL RATE|$)`)
	rsMatch := rsRe.FindStringSubmatch(text)
	rsSection := ""
	if len(rsMatch) >= 2 {
		rsSection = rsMatch[0]
	} else {
		rsSection = text
	}

	custRe := regexp.MustCompile(`Customer Charge:\s*\$?([0-9]+(?:\.[0-9]+)?)`)
	energyRe := regexp.MustCompile(`Energy Charge:\s*(\d+\.\d+|\.\d+|\d+)\$?\s*per kWh`)
	fuelRe := regexp.MustCompile(`TVA Fuel Charge:\s*(\d+\.\d+|\.\d+|\d+)\$?\s*per kWh`)

	customerCharge := parseFirstFloat(custRe, rsSection)
	energyRate := parseFirstFloat(energyRe, rsSection)
	fuelRate := parseFirstFloat(fuelRe, rsSection)

	energyCents := energyRate * 100
	fuelCents := fuelRate * 100

	now := time.Now().UTC()
	rawCopy := rsSection

	resp := &RatesResponse{
		Utility:   "CEMC",
		Source:    "CEMC Current Rates PDF",
		SourceURL: "https://cemc.org/my-account/#residential-rates",
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

func parseFirstFloat(re *regexp.Regexp, s string) float64 {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	var v float64
	fmt.Sscanf(m[1], "%f", &v)
	return v
}
