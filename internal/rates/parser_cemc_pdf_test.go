    package rates

    import "testing"

    func TestParseCEMCRatesFromText(t *testing.T) {
        sample := `
RESIDENTIAL RATE – SCHEDULE RS
(22) Customer Charge: $39.00 per month
Energy Charge: .08058$ per kWh per month
TVA Fuel Charge: .02177$ per kWh per month

SUPPLEMENTAL RESIDENTIAL RATE – SCHEDULE SRS
(21) Customer Charge:
`
        res, err := ParseCEMCRatesFromText(sample)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        rs := res.Rates.ResidentialStandard
        if !rs.IsPresent {
            t.Fatalf("expected residential standard to be present")
        }
        if rs.CustomerChargeMonthlyUSD != 39.0 {
            t.Errorf("unexpected customer charge: %v", rs.CustomerChargeMonthlyUSD)
        }
        if rs.EnergyRateUSDPerKWh <= 0 {
            t.Errorf("expected positive energy rate, got %v", rs.EnergyRateUSDPerKWh)
        }
        if rs.TVAFuelRateUSDPerKWh <= 0 {
            t.Errorf("expected positive fuel rate, got %v", rs.TVAFuelRateUSDPerKWh)
        }
    }
