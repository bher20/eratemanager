from cemc_rates.downloader import download_pdf
from cemc_rates.parser import parse_pdf


def test_parse_pdf_runs():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))

    assert "raw_text" in parsed
    assert isinstance(parsed["raw_text"], str)
    assert len(parsed["raw_text"]) > 100


def test_residential_standard_present():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    rs = parsed["residential_standard"]

    assert rs["is_present"] is True
    assert rs["customer_charge_usd_per_month"] == 39.00
    assert rs["energy_rate_usd_per_kwh"] == 0.08058
    assert rs["tva_fuel_rate_usd_per_kwh"] == 0.02177


def test_residential_supplemental_present():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    srs = parsed["residential_supplemental"]

    assert srs["is_present"] is True
    assert srs["customer_charge_part_a_usd_per_month"] == 28.00
    assert srs["customer_charge_part_b_usd_per_month"] == 40.54
    assert srs["energy_rate_usd_per_kwh"] == 0.08061
    assert srs["tva_fuel_rate_usd_per_kwh"] == 0.02177


def test_residential_seasonal_absent():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    seasonal = parsed["residential_seasonal"]

    assert seasonal["is_present"] is False
    assert seasonal["raw_section"] is None


def test_residential_tou_absent():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    tou = parsed["residential_tou"]

    assert tou["is_present"] is False
    assert tou["raw_section"] is None
