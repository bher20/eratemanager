package rates

import (
	"testing"
)

func TestProviders_DefaultsUsedWhenEnvEmpty(t *testing.T) {
	t.Setenv("ERATEMANAGER_PROVIDERS_JSON", "")
	ps := Providers()
	if len(ps) == 0 {
		t.Fatalf("expected non-empty default providers")
	}
	// Expect at least cemc and nes in defaults.
	foundCEMC := false
	foundNES := false
	for _, p := range ps {
		if p.Key == "cemc" {
			foundCEMC = true
		}
		if p.Key == "nes" {
			foundNES = true
		}
	}
	if !foundCEMC || !foundNES {
		t.Fatalf("expected default providers to include cemc and nes; got %+v", ps)
	}
}

func TestProviders_OverrideFromEnv(t *testing.T) {
	overrideJSON := `[
        {
            "key": "myutility",
            "name": "My Utility Power",
            "landingUrl": "https://myutility.example.com/rates/",
            "defaultPdfPath": "/data/myutility_rates.pdf",
            "notes": "Override provider"
        }
    ]`
	t.Setenv("ERATEMANAGER_PROVIDERS_JSON", overrideJSON)

	ps := Providers()
	if len(ps) != 1 {
		t.Fatalf("expected exactly 1 provider from override, got %d", len(ps))
	}
	if ps[0].Key != "myutility" {
		t.Fatalf("expected key 'myutility', got %q", ps[0].Key)
	}
	if ps[0].DefaultPDFPath != "/data/myutility_rates.pdf" {
		t.Fatalf("unexpected defaultPdfPath: %q", ps[0].DefaultPDFPath)
	}
}

func TestProviders_InvalidJSONFallsBack(t *testing.T) {
	t.Setenv("ERATEMANAGER_PROVIDERS_JSON", "{not valid json")
	ps := Providers()
	if len(ps) == 0 {
		t.Fatalf("expected fallback to defaults on invalid JSON")
	}
}

// Ensure GetProvider respects the current Providers() list.
func TestGetProvider_UsesOverride(t *testing.T) {
	overrideJSON := `[
        {
            "key": "x",
            "name": "X Utility",
            "landingUrl": "https://x.example.com",
            "defaultPdfPath": "/data/x_rates.pdf",
            "notes": ""
        }
    ]`
	t.Setenv("ERATEMANAGER_PROVIDERS_JSON", overrideJSON)

	p, ok := GetProvider("x")
	if !ok {
		t.Fatalf("expected provider 'x' to be found")
	}
	if p.Name != "X Utility" {
		t.Fatalf("unexpected provider name: %q", p.Name)
	}

	if _, ok := GetProvider("cemc"); ok {
		t.Fatalf("did not expect default provider cemc when override is set")
	}
}
