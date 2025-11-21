from eratemanager.downloader import download_pdf
from eratemanager.normalizer import normalize
from eratemanager.parser import parse_pdf


def test_end_to_end_pipeline():
    pdf_path = download_pdf(force=False)

    parsed = parse_pdf(str(pdf_path))
    normalized = normalize(parsed)

    assert normalized["utility"] == "CEMC"

    rates = normalized["rates"]
    assert rates["residential_standard"]["is_present"] is True
    assert rates["residential_supplemental"]["is_present"] is True


def test_end_to_end_contains_expected_fields():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))
    assert "residential_standard" in parsed
    assert "residential_supplemental" in parsed
