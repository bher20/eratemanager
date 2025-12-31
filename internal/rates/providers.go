package rates

import (
	"encoding/json"
	"os"
)

type ProviderDescriptor struct {
	Key            string `json:"key"`
	PDFAPIURL      string `json:"pdfApiUrl"`
	Name           string `json:"name"`
	LandingURL     string `json:"landingUrl"`
	DefaultPDFPath string `json:"defaultPdfPath"`
	Notes          string `json:"notes,omitempty"`
}

const providersEnv = "ERATEMANAGER_PROVIDERS_JSON"

func defaultProviders() []ProviderDescriptor {
	return withPDFURLs([]ProviderDescriptor{
		{
			Key:            "cemc",
			Name:           "Cumberland Electric Membership Corporation",
			LandingURL:     "https://cemc.org/my-account/#residential-rates",
			DefaultPDFPath: "/data/cemc_rates.pdf",
			Notes:          "CEMC residential rates",
		},
		{
			Key:            "nes",
			Name:           "Nashville Electric Service",
			LandingURL:     "https://www.nespower.com/rates/",
			DefaultPDFPath: "/data/nes_rates.pdf",
			Notes:          "NES residential rates",
		},
		{
			Key:            "kub",
			Name:           "Knoxville Utilities Board",
			LandingURL:     "https://www.kub.org/bills-payments/understand-your-bill/residential-rates/",
			DefaultPDFPath: "/data/kub_rates.pdf",
			Notes:          "KUB residential rates (TVA distributor)",
		},
	})
}

func Providers() []ProviderDescriptor {
	raw := os.Getenv(providersEnv)
	if raw == "" {
		return withPDFURLs(defaultProviders())
	}
	var out []ProviderDescriptor
	if err := json.Unmarshal([]byte(raw), &out); err != nil || len(out) == 0 {
		return withPDFURLs(defaultProviders())
	}
	return out
}

func GetProvider(key string) (ProviderDescriptor, bool) {
	for _, p := range Providers() {
		if p.Key == key {
			return p, true
		}
	}
	return ProviderDescriptor{}, false
}

func withPDFURLs(list []ProviderDescriptor) []ProviderDescriptor {
	for i := range list {
		if list[i].Key != "" {
			list[i].PDFAPIURL = "/rates/" + list[i].Key + "/pdf"
		}
	}
	return list
}
