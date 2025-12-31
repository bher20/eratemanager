package rates

import "time"

// WaterRatesResponse contains parsed water utility rate information
type WaterRatesResponse struct {
	ProviderKey  string    `json:"provider_key"`
	ProviderName string    `json:"provider_name"`
	FetchedAt    time.Time `json:"fetched_at"`

	// Water rates
	Water WaterRateDetails `json:"water"`

	// Sewer rates (optional, some providers bundle sewer)
	Sewer *SewerRateDetails `json:"sewer,omitempty"`
}

// WaterRateDetails contains the water-specific rate components
type WaterRateDetails struct {
	// MeterSizes maps meter size descriptions to their base charges
	// e.g., {"5/8 x 3/4 inch": 9.85, "1 inch": 13.37}
	MeterSizes map[string]float64 `json:"meter_sizes"`

	// DefaultMeterSize is the most common residential meter size
	DefaultMeterSize string `json:"default_meter_size"`

	// BaseCharge is the monthly base/service charge for the default meter
	BaseCharge float64 `json:"base_charge"`

	// UseRate is the per-unit usage charge
	UseRate float64 `json:"use_rate"`

	// UseRateUnit is the unit for the use rate (e.g., "gallon", "ccf", "1000 gallons")
	UseRateUnit string `json:"use_rate_unit"`

	// EffectiveDate when these rates became effective
	EffectiveDate string `json:"effective_date,omitempty"`
}

// SewerRateDetails contains sewer-specific rate components
type SewerRateDetails struct {
	// BaseCharge is the monthly base/service charge
	BaseCharge float64 `json:"base_charge"`

	// UseRate is the per-unit usage charge
	UseRate float64 `json:"use_rate"`

	// UseRateUnit is the unit for the use rate
	UseRateUnit string `json:"use_rate_unit"`

	// EffectiveDate when these rates became effective
	EffectiveDate string `json:"effective_date,omitempty"`
}

// CalculateWaterBill calculates the monthly water bill based on usage
func (w *WaterRatesResponse) CalculateWaterBill(gallons float64) float64 {
	waterCost := w.Water.BaseCharge + (gallons * w.Water.UseRate)

	sewerCost := 0.0
	if w.Sewer != nil {
		sewerCost = w.Sewer.BaseCharge + (gallons * w.Sewer.UseRate)
	}

	return waterCost + sewerCost
}

// CalculateWaterOnlyCost calculates just the water portion (no sewer)
func (w *WaterRatesResponse) CalculateWaterOnlyCost(gallons float64) float64 {
	return w.Water.BaseCharge + (gallons * w.Water.UseRate)
}

// CalculateSewerOnlyCost calculates just the sewer portion
func (w *WaterRatesResponse) CalculateSewerOnlyCost(gallons float64) float64 {
	if w.Sewer == nil {
		return 0
	}
	return w.Sewer.BaseCharge + (gallons * w.Sewer.UseRate)
}
