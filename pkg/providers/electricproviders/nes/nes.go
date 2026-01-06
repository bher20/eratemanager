package nes

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
	return "nes"
}

func (p *Provider) Name() string {
	return "Nashville Electric Service"
}

func (p *Provider) Type() providers.ProviderType {
	return providers.ProviderTypeElectric
}

func (p *Provider) LandingURL() string {
	return "https://www.nespower.com/rates"
}

func (p *Provider) DefaultPDFPath() string {
	return "rates_nes.pdf"
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
	// NES uses "Service Charge" instead of "Customer Charge"
	custRe := regexp.MustCompile(`(?:Customer|Service)\s+Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)

	// NES Energy Charge format
	energyCentsRe := regexp.MustCompile(`Energy Charge[:\s]*(?:Summer Period\s+)?([0-9]+(?:\.[0-9]+)?)\s*[¢c]`)
	energyUSDRe := regexp.MustCompile(`Energy Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*per kWh`)
	energyCentsAltRe := regexp.MustCompile(`Energy Charge[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)

	// Fuel adjustment (TVA)
	fuelRe := regexp.MustCompile(`Fuel(?: Cost)? Adjustment[:\s]*([0-9]+(?:\.[0-9]+)?)\s*[¢c]?(?:ents?)?\s*per kWh`)

	// TVA Grid Access Charge
	gridAccessRe := regexp.MustCompile(`(?:TVA )?Grid Access Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*per month`)

	customerCharge := shared.ParseFirstFloat(custRe, text)
	gridAccessCharge := shared.ParseFirstFloat(gridAccessRe, text)

	totalCustomerCharge := customerCharge
	if gridAccessCharge > 0 {
		totalCustomerCharge += gridAccessCharge
	}

	energyRate := 0.0
	if cents := shared.ParseFirstFloat(energyCentsRe, text); cents > 0 {
		energyRate = cents / 100.0
	} else if usd := shared.ParseFirstFloat(energyUSDRe, text); usd > 0 {
		energyRate = usd
	} else if cents := shared.ParseFirstFloat(energyCentsAltRe, text); cents > 0 {
		energyRate = cents / 100.0
	}

	fuelRate := 0.0
	if v := shared.ParseFirstFloat(fuelRe, text); v > 0 {
		if v < 1 {
			fuelRate = v
		} else {
			fuelRate = v / 100.0
		}
	}

	energyCents := energyRate * 100
	fuelCents := fuelRate * 100

	now := time.Now().UTC()
	rawCopy := text

	resp := &electricproviders.ElectricRatesResponse{
		Utility:   "Nashville Electric Service",
		Source:    "NES Rates PDF",
		SourceURL: "https://www.nespower.com/rates/",
		FetchedAt: now,
		ElectricRates: electricproviders.ElectricRates{
			ResidentialStandard: electricproviders.ResidentialStandard{
				IsPresent:                true,
				CustomerChargeMonthlyUSD: totalCustomerCharge,
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
