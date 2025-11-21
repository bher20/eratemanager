from cemc_rates.downloader import download_pdf
from cemc_rates.normalizer import normalize
from cemc_rates.parser import parse_pdf


def test_normalizer_structure():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    normalized = normalize(parsed)

    assert "utility" in normalized
    assert normalized["utility"] == "CEMC"
    assert "rates" in normalized

    rates = normalized["rates"]
    assert "residential_standard" in rates
    assert "residential_supplemental" in rates
    assert "residential_seasonal" in rates
    assert "residential_tou" in rates


def test_normalizer_residential_standard_values():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    normalized = normalize(parsed)

    rs = normalized["rates"]["residential_standard"]

    assert rs["is_present"] is True
    assert rs["customer_charge_monthly_usd"] == 39.00
    assert rs["energy_rate_usd_per_kwh"] == 0.08058
    assert rs["tva_fuel_rate_usd_per_kwh"] == 0.02177


def test_normalizer_residential_supplemental_values():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    normalized = normalize(parsed)

    srs = normalized["rates"]["residential_supplemental"]

    assert srs["is_present"] is True
    assert srs["customer_charge_part_a_monthly_usd"] == 28.00
    assert srs["customer_charge_part_b_monthly_usd"] == 40.54
    assert srs["energy_rate_usd_per_kwh"] == 0.08061
    assert srs["tva_fuel_rate_usd_per_kwh"] == 0.02177


def test_normalizer_placeholders():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    normalized = normalize(parsed)

    assert normalized["rates"]["residential_seasonal"]["is_present"] is False
    assert normalized["rates"]["residential_tou"]["is_present"] is False
