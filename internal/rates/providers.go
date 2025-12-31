package rates

import (
	"encoding/json"
	"os"
)

// ProviderType indicates the utility type
type ProviderType string

const (
	ProviderTypeElectric ProviderType = "electric"
	ProviderTypeWater    ProviderType = "water"
)

type ProviderDescriptor struct {
	Key            string       `json:"key"`
	Type           ProviderType `json:"type"`
	PDFAPIURL      string       `json:"pdfApiUrl,omitempty"`
	HTMLAPIURL     string       `json:"htmlApiUrl,omitempty"`
	Name           string       `json:"name"`
	LandingURL     string       `json:"landingUrl"`
	DefaultPDFPath string       `json:"defaultPdfPath,omitempty"`
	Notes          string       `json:"notes,omitempty"`
}

const providersEnv = "ERATEMANAGER_PROVIDERS_JSON"

func defaultProviders() []ProviderDescriptor {
	return []ProviderDescriptor{
		// Electric providers
		{
			Key:            "cemc",
			Type:           ProviderTypeElectric,
			Name:           "Cumberland Electric Membership Corporation",
			LandingURL:     "https://cemc.org/my-account/#residential-rates",
			DefaultPDFPath: "/data/cemc_rates.pdf",
			Notes:          "CEMC residential rates",
		},
		{
			Key:            "nes",
			Type:           ProviderTypeElectric,
			Name:           "Nashville Electric Service",
			LandingURL:     "https://www.nespower.com/rates/",
			DefaultPDFPath: "/data/nes_rates.pdf",
			Notes:          "NES residential rates",
		},
		{
			Key:            "kub",
			Type:           ProviderTypeElectric,
			Name:           "Knoxville Utilities Board",
			LandingURL:     "https://www.kub.org/bills-payments/understand-your-bill/residential-rates/",
			DefaultPDFPath: "/data/kub_rates.pdf",
			Notes:          "KUB residential rates (TVA distributor)",
		},
		// Water providers
		{
			Key:        "whud",
			Type:       ProviderTypeWater,
			Name:       "White House Utility District",
			LandingURL: "https://www.whud.org/rates-and-fees/",
			Notes:      "WHUD water and sewer rates",
		},
	}
}

func Providers() []ProviderDescriptor {
	raw := os.Getenv(providersEnv)
	if raw == "" {
		return withAPIURLs(defaultProviders())
	}
	var out []ProviderDescriptor
	if err := json.Unmarshal([]byte(raw), &out); err != nil || len(out) == 0 {
		return withAPIURLs(defaultProviders())
	}
	return out
}

// ElectricProviders returns only electric utility providers
func ElectricProviders() []ProviderDescriptor {
	var result []ProviderDescriptor
	for _, p := range Providers() {
		if p.Type == ProviderTypeElectric {
			result = append(result, p)
		}
	}
	return result
}

// WaterProviders returns only water utility providers
func WaterProviders() []ProviderDescriptor {
	var result []ProviderDescriptor
	for _, p := range Providers() {
		if p.Type == ProviderTypeWater {
			result = append(result, p)
		}
	}
	return result
}

func GetProvider(key string) (ProviderDescriptor, bool) {
	for _, p := range Providers() {
		if p.Key == key {
			return p, true
		}
	}
	return ProviderDescriptor{}, false
}

func withAPIURLs(list []ProviderDescriptor) []ProviderDescriptor {
	for i := range list {
		if list[i].Key != "" {
			switch list[i].Type {
			case ProviderTypeElectric:
				list[i].PDFAPIURL = "/rates/" + list[i].Key + "/pdf"
			case ProviderTypeWater:
				list[i].HTMLAPIURL = "/water/rates/" + list[i].Key
			}
		}
	}
	return list
}
