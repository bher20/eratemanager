package rates

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func init() {
	RegisterWaterParser(WaterParserConfig{
		Key:       "whud",
		Name:      "White House Utility District",
		ParseHTML: ParseWHUDRatesFromURL,
	})
}

// ParseWHUDRatesFromURL fetches the WHUD rates page and extracts water/sewer rates.
func ParseWHUDRatesFromURL(url string) (*WaterRatesResponse, error) {
	// WHUD server has a misconfigured SSL certificate chain (missing intermediate certs).
	// We use an insecure client as a workaround.
	client := InsecureHTTPClient()
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch WHUD rates page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WHUD rates page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read WHUD rates page: %w", err)
	}

	return ParseWHUDRatesFromHTML(string(body))
}

// ParseWHUDRatesFromHTML parses WHUD rates from raw HTML content.
// Based on the structure at https://www.whud.org/rates-and-fees/
func ParseWHUDRatesFromHTML(html string) (*WaterRatesResponse, error) {
	result := &WaterRatesResponse{
		ProviderKey:  "whud",
		ProviderName: "White House Utility District",
		FetchedAt:    time.Now(),
		Water: WaterRateDetails{
			MeterSizes:       make(map[string]float64),
			DefaultMeterSize: "5/8 x 3/4 inch",
			UseRateUnit:      "gallon",
		},
	}

	// Extract year from "2025 Water Rates" header
	yearRe := regexp.MustCompile(`(\d{4})\s+Water\s+Rates`)
	if match := yearRe.FindStringSubmatch(html); len(match) > 1 {
		result.Water.EffectiveDate = match[1]
	}

	// Extract water use rate: "Water Use Charge for all customers in 2025 is $0.00866/gallon"
	waterUseRe := regexp.MustCompile(`Water\s+Use\s+Charge[^$]*\$([0-9.]+)/gallon`)
	if match := waterUseRe.FindStringSubmatch(html); len(match) > 1 {
		if rate, err := strconv.ParseFloat(match[1], 64); err == nil {
			result.Water.UseRate = rate
		}
	}

	// Extract meter base rates
	// Pattern: "5/8" x 3/4" Meter $9.85" or similar
	meterRates := parseMeterRates(html)
	for size, rate := range meterRates {
		result.Water.MeterSizes[size] = rate
	}

	// Set default base charge from the standard residential meter
	if rate, ok := result.Water.MeterSizes["5/8 x 3/4 inch"]; ok {
		result.Water.BaseCharge = rate
	} else if len(result.Water.MeterSizes) > 0 {
		// Use the first/smallest meter size as default
		for _, rate := range result.Water.MeterSizes {
			result.Water.BaseCharge = rate
			break
		}
	}

	// Extract sewer rates
	// "Basic Service Charge: $10.49 per month"
	// "Sewer Use Charge: $0.01100 per gallon"
	sewerBaseRe := regexp.MustCompile(`(?i)sewer[^$]*Basic\s+Service\s+Charge[:\s]*\$([0-9.]+)`)
	sewerUseRe := regexp.MustCompile(`(?i)Sewer\s+Use\s+Charge[:\s]*\$([0-9.]+)\s+per\s+gallon`)

	// Try alternate pattern for sewer base
	if match := sewerBaseRe.FindStringSubmatch(html); len(match) > 1 {
		if rate, err := strconv.ParseFloat(match[1], 64); err == nil {
			if result.Sewer == nil {
				result.Sewer = &SewerRateDetails{UseRateUnit: "gallon"}
			}
			result.Sewer.BaseCharge = rate
		}
	} else {
		// Try simpler pattern: look in sewer section for base charge
		sewerSection := extractSewerSection(html)
		if sewerSection != "" {
			baseRe := regexp.MustCompile(`Basic\s+Service\s+Charge[:\s]*\$([0-9.]+)`)
			if match := baseRe.FindStringSubmatch(sewerSection); len(match) > 1 {
				if rate, err := strconv.ParseFloat(match[1], 64); err == nil {
					if result.Sewer == nil {
						result.Sewer = &SewerRateDetails{UseRateUnit: "gallon"}
					}
					result.Sewer.BaseCharge = rate
				}
			}
		}
	}

	if match := sewerUseRe.FindStringSubmatch(html); len(match) > 1 {
		if rate, err := strconv.ParseFloat(match[1], 64); err == nil {
			if result.Sewer == nil {
				result.Sewer = &SewerRateDetails{UseRateUnit: "gallon"}
			}
			result.Sewer.UseRate = rate
		}
	}

	// Set effective date for sewer if we have the year
	if result.Sewer != nil && result.Water.EffectiveDate != "" {
		result.Sewer.EffectiveDate = result.Water.EffectiveDate
	}

	// Validate we got the essential rates
	if result.Water.UseRate == 0 {
		return nil, fmt.Errorf("failed to parse water use rate from WHUD page")
	}

	return result, nil
}

// parseMeterRates extracts meter size to base charge mapping from HTML
func parseMeterRates(html string) map[string]float64 {
	rates := make(map[string]float64)

	// The HTML contains patterns like: "5/8" x 3/4" Meter $9.85"
	// We need to match the various meter sizes

	// Define meter patterns and their normalized names
	meterPatterns := []struct {
		pattern    string
		normalized string
	}{
		{`5/8"?\s*x\s*3/4"?\s*Meter\s*\$([0-9.]+)`, "5/8 x 3/4 inch"},
		{`(?:^|[^0-9])1"\s*Meter\s*\$([0-9.]+)`, "1 inch"},
		{`1\.5"\s*Meter\s*\$([0-9.]+)`, "1.5 inch"},
		{`(?:^|[^0-9])2"\s*Meter\s*\$([0-9.]+)`, "2 inch"},
		{`(?:^|[^0-9])3"\s*Meter\s*\$([0-9.]+)`, "3 inch"},
		{`(?:^|[^0-9])4"\s*Meter\s*\$([0-9.]+)`, "4 inch"},
		{`(?:^|[^0-9])6"\s*Meter\s*\$([0-9.]+)`, "6 inch"},
		{`(?:^|[^0-9])8"\s*Meter\s*\$([0-9.]+)`, "8 inch"},
		{`10"\s*Meter\s*\$([0-9.]+)`, "10 inch"},
	}

	for _, mp := range meterPatterns {
		re := regexp.MustCompile(mp.pattern)
		if match := re.FindStringSubmatch(html); len(match) >= 2 {
			if rate, err := strconv.ParseFloat(match[1], 64); err == nil {
				rates[mp.normalized] = rate
			}
		}
	}

	return rates
}

// extractSewerSection tries to extract just the sewer rates section from HTML
func extractSewerSection(html string) string {
	sewerRe := regexp.MustCompile(`(?is)WHUD\s+Sewer\s+Rates(.+?)(?:Other\s+Fees|$)`)
	if match := sewerRe.FindStringSubmatch(html); len(match) > 1 {
		return match[1]
	}
	return ""
}
