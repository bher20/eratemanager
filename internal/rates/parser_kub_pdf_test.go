package rates

import "testing"

func TestParseKubRatesFromText(t *testing.T) {
	sample := `
Customer Charge: $20.00 per month
Energy Charge: 11.34 cents per kWh
Fuel Cost Adjustment: 0.50 cents per kWh
`
	res, err := ParseKubRatesFromText(sample)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rs := res.Rates.ResidentialStandard
	if !rs.IsPresent {
		t.Fatalf("expected residential standard to be present")
	}
	if rs.CustomerChargeMonthlyUSD != 20.0 {
		t.Errorf("expected customer charge 20.0, got %v", rs.CustomerChargeMonthlyUSD)
	}
	if rs.EnergyRateUSDPerKWh <= 0 {
		t.Errorf("expected positive energy rate, got %v", rs.EnergyRateUSDPerKWh)
	}
}

func TestParseKubRatesFromText_TVAFormat(t *testing.T) {
	// Test format similar to other TVA distributors (NES-like)
	sample := `Schedule RS - Residential Service
Service Charge: $14.00 per month
TVA Grid Access Charge: $5.00 per month
Energy Charge: Summer 9.5¢ per kWh
TVA Fuel Cost Adjustment: 0.25¢ per kWh
`
	res, err := ParseKubRatesFromText(sample)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rs := res.Rates.ResidentialStandard
	if !rs.IsPresent {
		t.Fatalf("expected residential standard to be present")
	}
	// Service Charge ($14.00) + Grid Access ($5.00) = $19.00
	expectedCustomer := 19.0
	if rs.CustomerChargeMonthlyUSD < expectedCustomer-0.01 || rs.CustomerChargeMonthlyUSD > expectedCustomer+0.01 {
		t.Errorf("expected customer charge ~%v, got %v", expectedCustomer, rs.CustomerChargeMonthlyUSD)
	}
	// Energy rate should be 9.5 cents = 0.095 $/kWh
	expectedEnergy := 0.095
	if rs.EnergyRateUSDPerKWh < expectedEnergy-0.001 || rs.EnergyRateUSDPerKWh > expectedEnergy+0.001 {
		t.Errorf("expected energy rate ~%v, got %v", expectedEnergy, rs.EnergyRateUSDPerKWh)
	}
}

func TestParseKubRatesFromText_BasicServiceCharge(t *testing.T) {
	// KUB may use "Basic Service Charge" terminology
	sample := `
Basic Service Charge: $16.50 per month
Energy Charge: 10.25 cents per kWh
`
	res, err := ParseKubRatesFromText(sample)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rs := res.Rates.ResidentialStandard
	if rs.CustomerChargeMonthlyUSD != 16.5 {
		t.Errorf("expected customer charge 16.5, got %v", rs.CustomerChargeMonthlyUSD)
	}
}
