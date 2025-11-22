package rates

import (
    "context"
    "testing"
)

func TestGetResidential_CEMC_FallbackStub(t *testing.T) {
    svc := NewService(Config{}) // no PDF paths -> stub fallback
    ctx := context.Background()

    res, err := svc.GetResidential(ctx, "cemc")
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
    if rs.EnergyRateUSDPerKWh == 0 || rs.TVAFuelRateUSDPerKWh == 0 {
        t.Errorf("expected non-zero energy and fuel rates")
    }
}

func TestGetResidential_NES_FallbackStub(t *testing.T) {
    svc := NewService(Config{})
    ctx := context.Background()

    res, err := svc.GetResidential(ctx, "nes")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res.Utility != "NES" {
        t.Errorf("unexpected utility: %q", res.Utility)
    }
}

func TestGetResidential_Demo(t *testing.T) {
    svc := NewService(Config{})
    ctx := context.Background()

    res, err := svc.GetResidential(ctx, "demo")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res.Utility != "Demo Utility" {
        t.Errorf("unexpected utility: %q", res.Utility)
    }
}

func TestGetResidential_UnknownProvider(t *testing.T) {
    svc := NewService(Config{})
    ctx := context.Background()

    if _, err := svc.GetResidential(ctx, "unknown"); err == nil {
        t.Fatalf("expected error for unknown provider")
    }
}
