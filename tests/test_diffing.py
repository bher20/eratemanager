from cemc_rates.diffing import (
    extract_residential_summary,
    diff_summaries,
    format_changes_console,
    FieldChange
)

def test_extract_summary_structure():
    parsed = {
        "residential_standard": {
            "customer_charge_usd_per_month": 39.0,
            "energy_rate_usd_per_kwh": 0.08058,
            "tva_fuel_rate_usd_per_kwh": 0.02177
        },
        "residential_supplemental": {
            "customer_charge_part_a_usd_per_month": 28.0,
            "customer_charge_part_b_usd_per_month": 40.54,
            "energy_rate_usd_per_kwh": 0.08061,
            "tva_fuel_rate_usd_per_kwh": 0.02177
        }
    }

    summary = extract_residential_summary(parsed)

    assert summary["residential_standard"]["customer_charge"] == 39.0
    assert summary["residential_standard"]["energy_rate"] == 0.08058


def test_diff_detects_changes():
    baseline = {
        "residential_standard": {
            "customer_charge": 39.0,
            "energy_rate": 0.08058,
            "tva_fuel_rate": 0.02177,
        },
        "residential_supplemental": {
            "part_a": 28.0,
            "part_b": 40.54,
            "energy_rate": 0.08061,
            "tva_fuel_rate": 0.02177,
        }
    }

    current = {
        "residential_standard": {
            "customer_charge": 39.0,
            "energy_rate": 0.08200,       # changed
            "tva_fuel_rate": 0.02177,
        },
        "residential_supplemental": baseline["residential_supplemental"]
    }

    changes = diff_summaries(current, baseline)
    assert len(changes) == 1
    assert isinstance(changes[0], FieldChange)
    assert changes[0].field == "energy_rate"


def test_format_changes_console():
    ch = FieldChange(
        section="residential_standard",
        field="energy_rate",
        old=0.0805,
        new=0.0821
    )
    txt = format_changes_console([ch])

    assert "energy_rate" in txt
    assert "0.0805" in txt
    assert "0.0821" in txt
