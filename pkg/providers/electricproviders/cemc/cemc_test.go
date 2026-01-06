package cemc

import (
	"testing"
)

func TestParseText_Basic(t *testing.T) {
	sample := `RESIDENTIAL RATE SCHEDULE RS
Customer Charge: $39.00
Energy Charge: 0.09 per kWh
TVA Fuel Charge: 0.02 per kWh
`
	p := &Provider{}
	res, err := p.ParseText(sample)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil response")
	}
	if res.Utility != "Cumberland Electric Membership Corporation" {
		t.Errorf("unexpected utility: %q", res.Utility)
	}

	rs := res.ElectricRates.ResidentialStandard
	if rs.CustomerChargeMonthlyUSD != 39.0 {
		t.Errorf("unexpected customer charge: %v", rs.CustomerChargeMonthlyUSD)
	}
	if rs.EnergyRateUSDPerKWh != 0.09 {
		t.Errorf("unexpected energy rate: %v", rs.EnergyRateUSDPerKWh)
	}
}
