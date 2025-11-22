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
