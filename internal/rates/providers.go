package rates

import (
    "encoding/json"
    "os"
)

type ProviderDescriptor struct {
    Key            string `json:"key"`
    Name           string `json:"name"`
    LandingURL     string `json:"landingUrl"`
    DefaultPDFPath string `json:"defaultPdfPath"`
    Notes          string `json:"notes,omitempty"`
}

const providersEnv = "ERATEMANAGER_PROVIDERS_JSON"

func defaultProviders() []ProviderDescriptor {
    return []ProviderDescriptor{
        {
            Key:            "cemc",
            Name:           "CEMC",
            LandingURL:     "https://cemc.org/my-account/#residential-rates",
            DefaultPDFPath: "/data/cemc_rates.pdf",
            Notes:          "CEMC residential rates",
        },
        {
            Key:            "nes",
            Name:           "NES",
            LandingURL:     "https://www.nespower.com/rates/",
            DefaultPDFPath: "/data/nes_rates.pdf",
            Notes:          "NES residential rates",
        },
    }
}

func Providers() []ProviderDescriptor {
    raw := os.Getenv(providersEnv)
    if raw == "" {
        return defaultProviders()
    }
    var out []ProviderDescriptor
    if err := json.Unmarshal([]byte(raw), &out); err != nil || len(out) == 0 {
        return defaultProviders()
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
