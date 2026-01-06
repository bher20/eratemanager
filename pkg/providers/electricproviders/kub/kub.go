package kub

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
	return "kub"
}

func (p *Provider) Name() string {
	return "Knoxville Utilities Board"
}

func (p *Provider) Type() providers.ProviderType {
	return providers.ProviderTypeElectric
}

func (p *Provider) LandingURL() string {
	return "https://www.kub.org/bills-payments/understand-your-bill/residential-rates/"
}

func (p *Provider) DefaultPDFPath() string {
	return "rates_kub.pdf"
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
	basicServiceRe := regexp.MustCompile(`Basic Service Charge[:\s]*\$([0-9]+(?:\.[0-9]+)?)\s*per month`)
	custRe := regexp.MustCompile(`(?:Customer|Service)\s+Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)
	gridAccessRe := regexp.MustCompile(`(?:TVA )?Grid Access Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)\s*(?:per month)?`)

	summerRateRe := regexp.MustCompile(`Summer\s+Period\s+\$([0-9]+\.[0-9]+)\s*per kWh`)
	winterRateRe := regexp.MustCompile(`Winter\s+Period\s+\$([0-9]+\.[0-9]+)\s*per kWh`)
	transitionRateRe := regexp.MustCompile(`Transition\s+Period\s+\$([0-9]+\.[0-9]+)\s*per kWh`)

	energyCentsRe := regexp.MustCompile(`Energy Charge[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)
	energyCentSymbolRe := regexp.MustCompile(`Energy Charge[:\s]*(?:Summer\s+)?([0-9]+(?:\.[0-9]+)?)\s*[¢c]\s*per kWh`)
	energyUSDRe := regexp.MustCompile(`Energy Charge[:\s]*\$([0-9]+\.[0-9]+)\s*per kWh`)
	genericEnergyRe := regexp.MustCompile(`\$([0-9]+\.[0-9]{4,})\s*per kWh`)

	ppaRe := regexp.MustCompile(`Purchased Power Adjustment\s*\(([0-9]+(?:\.[0-9]+)?)\s*cents? per kWh\)`)
	fuelCentsRe := regexp.MustCompile(`(?:TVA )?Fuel(?: Cost)?\s*(?:Adjustment|Charge)[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)
	fuelCentSymbolRe := regexp.MustCompile(`(?:TVA )?Fuel(?: Cost)?\s*(?:Adjustment|Charge)[:\s]*([0-9]+(?:\.[0-9]+)?)\s*[¢c]\s*per kWh`)

	customerCharge := shared.ParseFirstFloat(basicServiceRe, text)
	if customerCharge == 0 {
		customerCharge = shared.ParseFirstFloat(custRe, text)
	}
	if gridAccess := shared.ParseFirstFloat(gridAccessRe, text); gridAccess > 0 {
		customerCharge += gridAccess
	}

	energyRate := 0.0
	if rate := shared.ParseFirstFloat(summerRateRe, text); rate > 0 {
		energyRate = rate
	} else if rate := shared.ParseFirstFloat(winterRateRe, text); rate > 0 {
		energyRate = rate
	} else if rate := shared.ParseFirstFloat(transitionRateRe, text); rate > 0 {
		energyRate = rate
	}

	if energyRate == 0 {
		if cents := shared.ParseFirstFloat(energyCentsRe, text); cents > 0 {
			energyRate = cents / 100.0
		} else if cents := shared.ParseFirstFloat(energyCentSymbolRe, text); cents > 0 {
			energyRate = cents / 100.0
		} else if usd := shared.ParseFirstFloat(energyUSDRe, text); usd > 0 {
			energyRate = usd
		} else if usd := shared.ParseFirstFloat(genericEnergyRe, text); usd > 0 {
			energyRate = usd
		}
	}

	fuelRate := 0.0
	if cents := shared.ParseFirstFloat(ppaRe, text); cents > 0 {
		fuelRate = cents / 100.0
	} else if cents := shared.ParseFirstFloat(fuelCentsRe, text); cents > 0 {
		fuelRate = cents / 100.0
	} else if cents := shared.ParseFirstFloat(fuelCentSymbolRe, text); cents > 0 {
		fuelRate = cents / 100.0
	}

	energyCents := energyRate * 100
	fuelCents := fuelRate * 100

	now := time.Now().UTC()
	rawCopy := text

	resp := &electricproviders.ElectricRatesResponse{
		Utility:   "Knoxville Utilities Board",
		Source:    "KUB Rates PDF",
		SourceURL: "https://www.kub.org/bills-payments/understand-your-bill/residential-rates/",
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
