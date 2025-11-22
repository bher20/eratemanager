    #!/usr/bin/env python3
    import argparse
    import textwrap
    from pathlib import Path

    ROOT = Path(__file__).resolve().parents[1]

    def main():
        ap = argparse.ArgumentParser(description="Scaffold a new utility provider for eRateManager (Go).")
        ap.add_argument("--key", required=True, help="Provider key slug (e.g. cemc, nes, myutility)")
        ap.add_argument("--name", required=True, help="Human-friendly provider name")
        ap.add_argument("--landing-url", required=True, help="Public landing/rates URL for this provider")
        ap.add_argument("--pdf-path", required=True, help="Default local cached PDF path, e.g. /data/myutility_rates.pdf")
        args = ap.parse_args()

        key = args.key.lower()
        name = args.name
        landing = args.landing_url
        pdf_path = args.pdf_path

        parser_file = ROOT / "internal" / "rates" / f"parser_{key}_pdf.go"
        test_file = ROOT / "internal" / "rates" / f"parser_{key}_pdf_test.go"

        parser_src = textwrap.dedent(f"""            package rates

            import (
                "bytes"
                "fmt"
                "io"
                "regexp"
                "time"

                pdf "github.com/ledongthuc/pdf"
            )

            // Parse{key.capitalize()}RatesFromPDF opens a {name} rates PDF at the given path,
            // extracts text, and delegates to Parse{key.capitalize()}RatesFromText.
            func Parse{key.capitalize()}RatesFromPDF(path string) (*RatesResponse, error) {{
                f, r, err := pdf.Open(path)
                if err != nil {{
                    return nil, fmt.Errorf("open pdf: %w", err)
                }}
                defer f.Close()

                rc, err := r.GetPlainText()
                if err != nil {{
                    return nil, fmt.Errorf("extract pdf text: %w", err)
                }}

                var buf bytes.Buffer
                if _, err := io.Copy(&buf, rc); err != nil {{
                    return nil, fmt.Errorf("read pdf text: %w", err)
                }}

                return Parse{key.capitalize()}RatesFromText(buf.String())
            }}

            // Parse{key.capitalize()}RatesFromText parses a plain-text representation of the
            // {name} residential rates and extracts fields using regex heuristics.
            //
            // TODO: Update the regexes below to match the actual PDF format.
            func Parse{key.capitalize()}RatesFromText(text string) (*RatesResponse, error) {{
                custRe := regexp.MustCompile(`Customer Charge[:\\s]*\\$?([0-9]+(?:\\.[0-9]+)?)`)
                energyUSDRe := regexp.MustCompile(`Energy Charge[:\\s]*\\$?([0-9]+(?:\\.[0-9]+)?)\\s*per kWh`)
                energyCentsRe := regexp.MustCompile(`Energy Charge[:\\s]*([0-9]+(?:\\.[0-9]+)?)\\s*cents?\\s*per kWh`)
                fuelRe := regexp.MustCompile(`Fuel(?: Cost)? Adjustment[:\\s]*([0-9]+(?:\\.[0-9]+)?)\\s*cents?\\s*per kWh`)

                customerCharge := parseFirstFloat(custRe, text)

                energyRate := parseFirstFloat(energyUSDRe, text)
                if energyRate == 0 {{
                    cents := parseFirstFloat(energyCentsRe, text)
                    energyRate = cents / 100.0
                }}

                fuelRate := 0.0
                if v := parseFirstFloat(fuelRe, text); v > 0 {{
                    fuelRate = v / 100.0
                }}

                energyCents := energyRate * 100
                fuelCents := fuelRate * 100

                now := time.Now().UTC()
                rawCopy := text

                resp := &RatesResponse{{
                    Utility:   "{name}",
                    Source:    "{name} Residential Rates PDF",
                    SourceURL: "{landing}",
                    FetchedAt: now,
                    Rates: Rates{{
                        ResidentialStandard: ResidentialStandard{{
                            IsPresent:                true,
                            CustomerChargeMonthlyUSD: customerCharge,
                            EnergyRateUSDPerKWh:      energyRate,
                            EnergyRateCentsPerKWh:    energyCents,
                            TVAFuelRateUSDPerKWh:     fuelRate,
                            TVAFuelRateCentsPerKWh:   fuelCents,
                            RawSection:               &rawCopy,
                        }},
                    }},
                }}
                return resp, nil
            }}
        """)

        test_src = textwrap.dedent(f"""            package rates

            import "testing"

            func TestParse{key.capitalize()}RatesFromText(t *testing.T) {{
                sample := `
Customer Charge: $20.00 per month
Energy Charge: 11.34 cents per kWh
Fuel Cost Adjustment: 0.50 cents per kWh
`
                res, err := Parse{key.capitalize()}RatesFromText(sample)
                if err != nil {{
                    t.Fatalf("unexpected error: %v", err)
                }}
                rs := res.Rates.ResidentialStandard
                if !rs.IsPresent {{
                    t.Fatalf("expected residential standard to be present")
                }}
                if rs.CustomerChargeMonthlyUSD <= 0 {{
                    t.Errorf("expected positive customer charge, got %v", rs.CustomerChargeMonthlyUSD)
                }}
                if rs.EnergyRateUSDPerKWh <= 0 {{
                    t.Errorf("expected positive energy rate, got %v", rs.EnergyRateUSDPerKWh)
                }}
            }}
        """)

        parser_file.write_text(parser_src, encoding="utf-8")
        test_file.write_text(test_src, encoding="utf-8")

        print("Scaffolded provider:")
        print(f"  key        : {key}")
        print(f"  name       : {name}")
        print(f"  landingURL : {landing}")
        print(f"  pdfPath    : {pdf_path}")
        print()
        print("Next steps:")
        print("1) Add a provider descriptor to internal/rates/providers.go:")
        print()
        print(f"   {{")
        print(f"       Key:           \"{key}\",")
        print(f"       Name:          \"{name}\",")
        print(f"       LandingURL:    \"{landing}\",")
        print(f"       DefaultPDFPath: \"{pdf_path}\",")
        print(f"       Notes:         \"{name} residential rates.\",")
        print(f"   }},")
        print()
        print("2) Add a switch case in internal/rates/service.go in GetResidential:")
        print()
        print(f"   case \"{key}\":")
        print(f"       // Implement get{key.capitalize()}Residential similar to CEMC/NES")
        print(f"       return s.get{key.capitalize()}Residential(ctx)")
        print()
        print("3) Implement get{key.capitalize()}Residential(ctx) in service.go, mirroring CEMC/NES logic.")
        print("4) Add a provider entry to the Home Assistant integration's const.py:")
        print()
        print(f"   \"{key}\": {{")
        print(f"       \"name\": \"{name}\",")
        print(f"       \"default_url\": \"https://<your-hostname>/rates/{key}/residential\",")
        print(f"   }},")
        print()
        print("5) Add a pdf URL entry in Helm values.yaml and (optionally) a cronjob template to keep the PDF fresh.")
        print()
        print("Finally:")
        print("   gofmt -w internal/rates/parser_{key}_pdf.go internal/rates/parser_{key}_pdf_test.go")
        print("   go test ./...")

    if __name__ == "__main__":
        main()
