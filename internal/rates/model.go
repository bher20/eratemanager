package rates

import "time"

// RatesResponse matches the JSON schema used by the original Python service
// and consumed by the Home Assistant integration.
type RatesResponse struct {
    Utility   string    `json:"utility"`
    Source    string    `json:"source"`
    SourceURL string    `json:"source_url"`
    FetchedAt time.Time `json:"fetched_at"`
    Rates     Rates     `json:"rates"`
}

type Rates struct {
    ResidentialStandard     ResidentialStandard     `json:"residential_standard"`
    ResidentialSupplemental ResidentialSupplemental `json:"residential_supplemental"`
    ResidentialSeasonal     ResidentialSeasonal     `json:"residential_seasonal"`
    ResidentialTOU          ResidentialTOU          `json:"residential_tou"`
}

type ResidentialStandard struct {
    IsPresent                bool    `json:"is_present"`
    CustomerChargeMonthlyUSD float64 `json:"customer_charge_monthly_usd"`
    EnergyRateUSDPerKWh      float64 `json:"energy_rate_usd_per_kwh"`
    EnergyRateCentsPerKWh    float64 `json:"energy_rate_cents_per_kwh"`
    TVAFuelRateUSDPerKWh     float64 `json:"tva_fuel_rate_usd_per_kwh"`
    TVAFuelRateCentsPerKWh   float64 `json:"tva_fuel_rate_cents_per_kwh"`
    RawSection               *string `json:"raw_section"`
}

type ResidentialSupplemental struct {
    IsPresent                     bool    `json:"is_present"`
    CustomerChargePartAMonthlyUSD float64 `json:"customer_charge_part_a_monthly_usd"`
    CustomerChargePartBMonthlyUSD float64 `json:"customer_charge_part_b_monthly_usd"`
    EnergyRateUSDPerKWh           float64 `json:"energy_rate_usd_per_kwh"`
    EnergyRateCentsPerKWh         float64 `json:"energy_rate_cents_per_kwh"`
    TVAFuelRateUSDPerKWh          float64 `json:"tva_fuel_rate_usd_per_kwh"`
    TVAFuelRateCentsPerKWh        float64 `json:"tva_fuel_rate_cents_per_kwh"`
    RawSection                    *string `json:"raw_section"`
}

type ResidentialSeasonal struct {
    IsPresent           bool      `json:"is_present"`
    RawSection          *string   `json:"raw_section"`
    SummerRateUSDPerKWh *float64  `json:"summer_rate_usd_per_kwh"`
    WinterRateUSDPerKWh *float64  `json:"winter_rate_usd_per_kwh"`
    SummerMonths        []string  `json:"summer_months"`
    WinterMonths        []string  `json:"winter_months"`
}

type ResidentialTOU struct {
    IsPresent             bool      `json:"is_present"`
    RawSection            *string   `json:"raw_section"`
    OnPeakRateUSDPerKWh   *float64  `json:"on_peak_rate_usd_per_kwh"`
    OffPeakRateUSDPerKWh  *float64  `json:"off_peak_rate_usd_per_kwh"`
    ShoulderRateUSDPerKWh *float64  `json:"shoulder_rate_usd_per_kwh"`
    OnPeakHours           []string  `json:"on_peak_hours"`
    OffPeakHours          []string  `json:"off_peak_hours"`
    ShoulderHours         []string  `json:"shoulder_hours"`
}
