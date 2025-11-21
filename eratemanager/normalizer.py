from datetime import datetime, timezone
from typing import Any, Dict


def _cents(val: float | None) -> float | None:
    return round(val * 100, 5) if val is not None else None


def normalize(parsed: Dict[str, Any]) -> Dict[str, Any]:
    """
    Normalize parsed values into a stable JSON schema.
    """
    now = datetime.now(timezone.utc).isoformat()

    rs = parsed.get("residential_standard", {}) or {}
    srs = parsed.get("residential_supplemental", {}) or {}
    seasonal = parsed.get("residential_seasonal", {}) or {}
    tou = parsed.get("residential_tou", {}) or {}

    return {
        "utility": "CEMC",
        "source": "CEMC Current Rates PDF",
        "source_url": "https://cemc.org/my-account/#residential-rates",
        "fetched_at": now,
        "rates": {
            "residential_standard": {
                "is_present": rs.get("is_present", False),
                "customer_charge_monthly_usd": rs.get("customer_charge_usd_per_month"),

                # Dollars + Cents
                "energy_rate_usd_per_kwh": rs.get("energy_rate_usd_per_kwh"),
                "energy_rate_cents_per_kwh": _cents(rs.get("energy_rate_usd_per_kwh")),
                "tva_fuel_rate_usd_per_kwh": rs.get("tva_fuel_rate_usd_per_kwh"),
                "tva_fuel_rate_cents_per_kwh": _cents(rs.get("tva_fuel_rate_usd_per_kwh")),

                "raw_section": rs.get("raw_section"),
            },

            "residential_supplemental": {
                "is_present": srs.get("is_present", False),
                "customer_charge_part_a_monthly_usd": srs.get("customer_charge_part_a_usd_per_month"),
                "customer_charge_part_b_monthly_usd": srs.get("customer_charge_part_b_usd_per_month"),

                # Dollars + Cents
                "energy_rate_usd_per_kwh": srs.get("energy_rate_usd_per_kwh"),
                "energy_rate_cents_per_kwh": _cents(srs.get("energy_rate_usd_per_kwh")),
                "tva_fuel_rate_usd_per_kwh": srs.get("tva_fuel_rate_usd_per_kwh"),
                "tva_fuel_rate_cents_per_kwh": _cents(srs.get("tva_fuel_rate_usd_per_kwh")),

                "raw_section": srs.get("raw_section"),
            },

            "residential_seasonal": seasonal,
            "residential_tou": tou,
        },
    }
