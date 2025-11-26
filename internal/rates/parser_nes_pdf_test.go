package rates

import "testing"

func TestParseNESRatesFromText(t *testing.T) {
	sample := `
Residential Service
Customer Charge: $20.00 per month
Energy Charge: 11.34 cents per kWh
Fuel Cost Adjustment: 0.50 cents per kWh
`
	res, err := ParseNESRatesFromText(sample)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rs := res.Rates.ResidentialStandard
	if !rs.IsPresent {
		t.Fatalf("expected residential standard to be present")
	}
	if rs.CustomerChargeMonthlyUSD != 20.0 {
		t.Errorf("unexpected customer charge: %v", rs.CustomerChargeMonthlyUSD)
	}
	if rs.EnergyRateUSDPerKWh <= 0 {
		t.Errorf("expected positive energy rate, got %v", rs.EnergyRateUSDPerKWh)
	}
	if rs.TVAFuelRateUSDPerKWh <= 0 {
		t.Errorf("expected positive fuel rate, got %v", rs.TVAFuelRateUSDPerKWh)
	}
}

func TestParseNESRatesFromText_ActualPDFFormat(t *testing.T) {
	// This matches the actual NES PDF format
	sample := `Schedule RS Page 1 of 2 ELECTRIC POWER BOARD OF THE METROPOLITAN GOVERNMENT
Base Charges
Service Charge: $14.06 per month if the customer's highest monthly kWh usage during the latest 12-month period is not more than 500 kWh
TVA Grid Access Charge: $4.50 per month if the customer's average monthly kWh usage during the latest 12-month period is not more than 500 kWh
Energy Charge: Summer Period 9.254¢ per kWh per month (including the additional hydro charge amount of 0.186¢)
Winter Period 8.889¢ per kWh per month
Transition Period 8.664¢ per kWh per month
`
	res, err := ParseNESRatesFromText(sample)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rs := res.Rates.ResidentialStandard
	if !rs.IsPresent {
		t.Fatalf("expected residential standard to be present")
	}
	// Service Charge ($14.06) + Grid Access ($4.50) = $18.56
	expectedCustomer := 18.56
	if rs.CustomerChargeMonthlyUSD < expectedCustomer-0.01 || rs.CustomerChargeMonthlyUSD > expectedCustomer+0.01 {
		t.Errorf("expected customer charge ~%v, got %v", expectedCustomer, rs.CustomerChargeMonthlyUSD)
	}
	// Energy rate should be 9.254 cents = 0.09254 $/kWh
	expectedEnergy := 0.09254
	if rs.EnergyRateUSDPerKWh < expectedEnergy-0.0001 || rs.EnergyRateUSDPerKWh > expectedEnergy+0.0001 {
		t.Errorf("expected energy rate ~%v, got %v", expectedEnergy, rs.EnergyRateUSDPerKWh)
	}
	// Energy cents should be ~9.254
	if rs.EnergyRateCentsPerKWh < 9.25 || rs.EnergyRateCentsPerKWh > 9.26 {
		t.Errorf("expected energy rate cents ~9.254, got %v", rs.EnergyRateCentsPerKWh)
	}
}
