package rates

import (
	"context"
	"testing"
)

// TestParseCEMCRatesFromText_Basic tests the CEMC parser with sample text.
func TestParseCEMCRatesFromText_Basic(t *testing.T) {
	sample := `RESIDENTIAL RATE SCHEDULE RS
Customer Charge: $39.00
Energy Charge: 0.09 per kWh
TVA Fuel Charge: 0.02 per kWh
`
	res, err := ParseCEMCRatesFromText(sample)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil response")
	}
	if res.Utility != "CEMC" {
		t.Errorf("unexpected utility: %q", res.Utility)
	}

	rs := res.Rates.ResidentialStandard
	if !rs.IsPresent {
		t.Errorf("expected residential_standard to be present")
	}
	if rs.CustomerChargeMonthlyUSD != 39.0 {
		t.Errorf("unexpected customer charge: %v", rs.CustomerChargeMonthlyUSD)
	}
}

// TestParserRegistry ensures parsers are registered via init().
func TestParserRegistry(t *testing.T) {
	// CEMC and NES should be registered
	keys := ListParsers()
	if len(keys) < 2 {
		t.Errorf("expected at least 2 parsers registered, got %d", len(keys))
	}

	cemc, ok := GetParser("cemc")
	if !ok {
		t.Fatalf("cemc parser not registered")
	}
	if cemc.Name == "" {
		t.Error("cemc parser has empty name")
	}
	if cemc.ParsePDF == nil {
		t.Error("cemc parser has nil ParsePDF")
	}

	nes, ok := GetParser("nes")
	if !ok {
		t.Fatalf("nes parser not registered")
	}
	if nes.Name == "" {
		t.Error("nes parser has empty name")
	}
}

// TestGetResidential_UnknownProvider ensures unknown providers return an error.
func TestGetResidential_UnknownProvider(t *testing.T) {
	svc := NewService(Config{})
	ctx := context.Background()

	if _, err := svc.GetResidential(ctx, "unknown"); err == nil {
		t.Fatalf("expected error for unknown provider")
	}
}
