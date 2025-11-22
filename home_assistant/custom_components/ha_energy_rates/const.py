DOMAIN = "ha_energy_rates"

PROVIDERS = {
    "cemc": {
        "name": "CEMC",
        "default_url": "https://rates.example.com/rates/cemc/residential",
    },
    "nes": {
        "name": "NES",
        "default_url": "https://rates.example.com/rates/nes/residential",
    },
    "demo": {
        "name": "Demo Utility",
        "default_url": "https://rates.example.com/rates/demo/residential",
    },
}

DEFAULT_PROVIDER = "cemc"
DEFAULT_SCAN_INTERVAL = 3600
