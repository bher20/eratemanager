import json
import subprocess
import sys
from pathlib import Path

def run_cli(args):
    """Run the installed CLI and return stdout."""
    cmd = [sys.executable, "-m", "cemc_rates.cli"] + args
    out = subprocess.check_output(cmd, text=True)
    return out

def test_cli_residential_outputs_cents(monkeypatch):
    # Fake parsed+normalized data
    sample_json = {
        "residential_standard": {
            "customer_charge_monthly_usd": 39.0,
            "energy_rate_usd_per_kwh": 0.08058,
            "energy_rate_cents_per_kwh": 8.058,
            "tva_fuel_rate_usd_per_kwh": 0.02177,
            "tva_fuel_rate_cents_per_kwh": 2.177
        }
    }

    # Mock normalization pipeline
    monkeypatch.setenv("CEMC_RATES_TEST_MODE", "1")
    monkeypatch.setattr(
        "cemc_rates.cli._load_current_parsed",
        lambda *a, **kw: {"fake": True}
    )
    monkeypatch.setattr(
        "cemc_rates.cli.normalize",
        lambda *a, **kw: {"rates": sample_json}
    )

    # Run CLI
    out = run_cli(["fetch", "--residential"])
    data = json.loads(out)

    assert "energy_rate_cents_per_kwh" in data["residential_standard"]
    assert data["residential_standard"]["energy_rate_cents_per_kwh"] == 8.058
