package cemc

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/bher20/eratemanager/pkg/providers"
	"github.com/bher20/eratemanager/pkg/providers/electricproviders"
	"github.com/bher20/eratemanager/pkg/providers/shared"
	"github.com/ledongthuc/pdf"
)

func init() {
	electricproviders.Register(&Provider{})
}

type Provider struct{}

func (p *Provider) Key() string {
	return "cemc"
}

func (p *Provider) Name() string {
	return "Cumberland Electric Membership Corporation"
}

func (p *Provider) Type() providers.ProviderType {
	return providers.ProviderTypeElectric
}

func (p *Provider) LandingURL() string {
	return "https://cemc.org/my-account/#residential-rates"
}

func (p *Provider) DefaultPDFPath() string {
	return "rates_cemc.pdf"
}

func (p *Provider) ParsePDF(path string) (*electricproviders.ElectricRatesResponse, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	rc, err := r.GetPlainText()
	if err != nil {
		return nil, fmt.Errorf("extract pdf text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rc); err != nil {
		return nil, fmt.Errorf("read pdf text: %w", err)
	}

	return p.ParseText(buf.String())
}

func (p *Provider) ParseText(text string) (*electricproviders.ElectricRatesResponse, error) {
	// Try to narrow to the residential RS section.
	rsRe := regexp.MustCompile(`RESIDENTIAL RATE[^\n]*SCHEDULE RS(?s)(.+?)(?:SUPPLEMENTAL RESIDENTIAL RATE|$)`)
	rsMatch := rsRe.FindStringSubmatch(text)
	rsSection := ""
	if len(rsMatch) >= 2 {
		rsSection = rsMatch[0]
	} else {
		rsSection = text
	}

	custRe := regexp.MustCompile(`Customer Charge:\s*\$?([0-9]+(?:\.[0-9]+)?)`)
	energyRe := regexp.MustCompile(`Energy Charge:\s*(\d+\.\d+|\.\d+|\d+)\$?\s*per kWh`)
	fuelRe := regexp.MustCompile(`TVA Fuel Charge:\s*(\d+\.\d+|\.\d+|\d+)\$?\s*per kWh`)

	customerCharge := shared.ParseFirstFloat(custRe, rsSection)
	energyRate := shared.ParseFirstFloat(energyRe, rsSection)
	fuelRate := shared.ParseFirstFloat(fuelRe, rsSection)

	energyCents := energyRate * 100
	fuelCents := fuelRate * 100

	now := time.Now().UTC()
	rawCopy := rsSection

	resp := &electricproviders.ElectricRatesResponse{
		Utility:   "Cumberland Electric Membership Corporation",
		Source:    "CEMC Current Rates PDF",
		SourceURL: "https://cemc.org/my-account/#residential-rates",
		FetchedAt: now,
		ElectricRates: electricproviders.ElectricRates{
			ResidentialStandard: electricproviders.ResidentialStandard{
				IsPresent:                true,
				CustomerChargeMonthlyUSD: customerCharge,
				EnergyRateUSDPerKWh:      energyRate,
				EnergyRateCentsPerKWh:    energyCents,
				TVAFuelRateUSDPerKWh:     fuelRate,
				TVAFuelRateCentsPerKWh:   fuelCents,
				RawSection:               &rawCopy,
			},
		},
	}

	return resp, nil
}
