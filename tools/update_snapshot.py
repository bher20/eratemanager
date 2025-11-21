"""
Helper script to regenerate the residential rate snapshot
from the current CEMC PDF.

Run from the project root:

    python tools/update_snapshot.py
"""

import json
from pathlib import Path

from eratemanager.downloader import download_pdf
from eratemanager.parser import parse_pdf


SNAPSHOT_PATH = Path("snapshots/residential_v1.json")


def main() -> None:
    pdf_path = download_pdf(force=False)
    parsed = parse_pdf(str(pdf_path))

    rs = parsed["residential_standard"]
    srs = parsed["residential_supplemental"]

    new_snapshot = {
        "residential_standard": {
            "customer_charge": rs["customer_charge_usd_per_month"],
            "energy_rate": rs["energy_rate_usd_per_kwh"],
            "tva_fuel_rate": rs["tva_fuel_rate_usd_per_kwh"],
        },
        "residential_supplemental": {
            "part_a": srs["customer_charge_part_a_usd_per_month"],
            "part_b": srs["customer_charge_part_b_usd_per_month"],
            "energy_rate": srs["energy_rate_usd_per_kwh"],
            "tva_fuel_rate": srs["tva_fuel_rate_usd_per_kwh"],
        },
        "metadata": {
            "pdf_version_label": "CEMC_2025_Nov_v2",
        },
    }

    SNAPSHOT_PATH.parent.mkdir(parents=True, exist_ok=True)
    SNAPSHOT_PATH.write_text(json.dumps(new_snapshot, indent=2))
    print(f"Snapshot updated at {SNAPSHOT_PATH}")


if __name__ == "__main__":  # pragma: no cover
    main()
