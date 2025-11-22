DOMAIN = "ha_energy_rates"

PROVIDERS = {
    "cemc": {
        "name": "CEMC",
        "default_url": "https://rates.bherville.com/rates/cemc/residential",
    },
    "nes": {
        "name": "NES",
        "default_url": "https://rates.bherville.com/rates/nes/residential",
    },
    "demo": {
        "name": "Demo Utility",
        "default_url": "https://rates.bherville.com/rates/demo/residential",
    },
}

DEFAULT_PROVIDER = "cemc"
DEFAULT_SCAN_INTERVAL = 3600
