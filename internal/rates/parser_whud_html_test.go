package rates

import (
	"testing"
)

// Sample HTML content based on https://www.whud.org/rates-and-fees/
const sampleWHUDHTML = `
<html>
<body>
<h2>2025 Water Rates</h2>
<h3>Meter Base Rates:</h3>
<p>5/8" x 3/4" Meter $9.85 1" Meter $13.37 1.5" Meter $32.99 2" Meter $48.58 3" Meter $82.28 4" Meter $88.82 6" Meter $142.64 8" Meter $219.10 10" Meter $275.85</p>
<p>The Water Use Charge for all customers in 2025 is $0.00866/gallon.</p>

<h2>2025 WHUD Sewer Rates</h2>
<p>Basic Service Charge: $10.49 per month</p>
<p>Sewer Use Charge: $0.01100 per gallon</p>

<h3>Examples</h3>
<h5>Meter using 2,000 gallons per month (5/8 inch x 3/4 inch meter):</h5>
<p>Basic Service Charge: $9.85 Water Use Charge (2,000 x .00866) $17.32 $27.17/month</p>
</body>
</html>
`

func TestParseWHUDRatesFromHTML(t *testing.T) {
	result, err := ParseWHUDRatesFromHTML(sampleWHUDHTML)
	if err != nil {
		t.Fatalf("ParseWHUDRatesFromHTML failed: %v", err)
	}

	// Check provider info
	if result.ProviderKey != "whud" {
		t.Errorf("expected provider_key 'whud', got %q", result.ProviderKey)
	}
	if result.ProviderName != "White House Utility District" {
		t.Errorf("expected provider_name 'White House Utility District', got %q", result.ProviderName)
	}

	// Check water rates
	if result.Water.UseRate != 0.00866 {
		t.Errorf("expected water use rate 0.00866, got %f", result.Water.UseRate)
	}
	if result.Water.UseRateUnit != "gallon" {
		t.Errorf("expected water use rate unit 'gallon', got %q", result.Water.UseRateUnit)
	}
	if result.Water.EffectiveDate != "2025" {
		t.Errorf("expected effective date '2025', got %q", result.Water.EffectiveDate)
	}

	// Check base charge for default meter
	if result.Water.BaseCharge != 9.85 {
		t.Errorf("expected base charge 9.85, got %f", result.Water.BaseCharge)
	}

	// Check meter sizes
	expectedMeters := map[string]float64{
		"5/8 x 3/4 inch": 9.85,
		"1 inch":         13.37,
	}
	for size, expectedRate := range expectedMeters {
		if rate, ok := result.Water.MeterSizes[size]; !ok {
			t.Errorf("missing meter size %q", size)
		} else if rate != expectedRate {
			t.Errorf("expected meter %q rate %f, got %f", size, expectedRate, rate)
		}
	}

	// Check sewer rates
	if result.Sewer == nil {
		t.Fatal("expected sewer rates to be parsed")
	}
	if result.Sewer.BaseCharge != 10.49 {
		t.Errorf("expected sewer base charge 10.49, got %f", result.Sewer.BaseCharge)
	}
	if result.Sewer.UseRate != 0.011 {
		t.Errorf("expected sewer use rate 0.011, got %f", result.Sewer.UseRate)
	}
}

func TestCalculateWaterBill(t *testing.T) {
	rates := &WaterRatesResponse{
		Water: WaterRateDetails{
			BaseCharge:  9.85,
			UseRate:     0.00866,
			UseRateUnit: "gallon",
		},
		Sewer: &SewerRateDetails{
			BaseCharge:  10.49,
			UseRate:     0.011,
			UseRateUnit: "gallon",
		},
	}

	// Test case from WHUD website: 2,000 gallons
	// Water: $9.85 + (2000 * 0.00866) = $9.85 + $17.32 = $27.17
	// Sewer: $10.49 + (2000 * 0.011) = $10.49 + $22.00 = $32.49
	// Total: $59.66

	waterOnly := rates.CalculateWaterOnlyCost(2000)
	expectedWater := 27.17
	if waterOnly < expectedWater-0.01 || waterOnly > expectedWater+0.01 {
		t.Errorf("expected water cost ~%f, got %f", expectedWater, waterOnly)
	}

	sewerOnly := rates.CalculateSewerOnlyCost(2000)
	expectedSewer := 32.49
	if sewerOnly < expectedSewer-0.01 || sewerOnly > expectedSewer+0.01 {
		t.Errorf("expected sewer cost ~%f, got %f", expectedSewer, sewerOnly)
	}

	total := rates.CalculateWaterBill(2000)
	expectedTotal := 59.66
	if total < expectedTotal-0.02 || total > expectedTotal+0.02 {
		t.Errorf("expected total ~%f, got %f", expectedTotal, total)
	}
}

func TestWaterParserRegistration(t *testing.T) {
	parser, ok := GetWaterParser("whud")
	if !ok {
		t.Fatal("WHUD water parser not registered")
	}
	if parser.Name != "White House Utility District" {
		t.Errorf("expected parser name 'White House Utility District', got %q", parser.Name)
	}
	if parser.ParseHTML == nil {
		t.Error("ParseHTML function is nil")
	}
}
