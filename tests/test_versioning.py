import json
from pathlib import Path

from cemc_rates.downloader import download_pdf
from cemc_rates.parser import parse_pdf


SNAPSHOT_PATH = Path("snapshots/residential_v1.json")


def load_snapshot() -> dict:
    assert SNAPSHOT_PATH.exists(), (
        "Snapshot missing — expected snapshots/residential_v1.json\n"
        "If CEMC released a new rate sheet or this is the first run, "
        "generate a snapshot with tools/update_snapshot.py."
    )
    with open(SNAPSHOT_PATH, "r") as f:
        return json.load(f)


def extract_residential_summary(parsed: dict) -> dict:
    """Return only the fields that should remain stable across versions."""
    rs = parsed["residential_standard"]
    srs = parsed["residential_supplemental"]

    return {
        "residential_standard": {
            "customer_charge": rs.get("customer_charge_usd_per_month"),
            "energy_rate": rs.get("energy_rate_usd_per_kwh"),
            "tva_fuel_rate": rs.get("tva_fuel_rate_usd_per_kwh"),
        },
        "residential_supplemental": {
            "part_a": srs.get("customer_charge_part_a_usd_per_month"),
            "part_b": srs.get("customer_charge_part_b_usd_per_month"),
            "energy_rate": srs.get("energy_rate_usd_per_kwh"),
            "tva_fuel_rate": srs.get("tva_fuel_rate_usd_per_kwh"),
        },
    }


def test_pdf_versioning():
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))

    snapshot = load_snapshot()
    current = extract_residential_summary(parsed)

    expected = {
        "residential_standard": snapshot["residential_standard"],
        "residential_supplemental": snapshot["residential_supplemental"],
    }

    assert current == expected, (
        "PDF version mismatch!\n"
        "The parsed rates differ from the last known snapshot.\n"
        "If CEMC released a new rate sheet, review the changes and update the snapshot."
    )
