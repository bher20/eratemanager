# 📘 eRateManager
*A unified platform for extracting, normalizing, and serving residential energy rates.*

eRateManager is a Kubernetes-ready, API-driven service that automatically downloads utility rate documents (PDF), parses the rate structures, normalizes them into consistent machine-readable JSON, and exposes them through a clean REST API.

Originally built for CEMC (TN) energy rates, eRateManager is designed to scale to many utilities and feed Home Assistant, dashboards, or billing systems.

---

## 🚀 Features

### 🔍 Automated Rate Extraction
- Downloads official rate PDFs on a schedule
- Parses:
  - Residential standard rates
  - Fuel cost adjustments
  - Supplemental rates
  - Seasonal rates (summer/winter)
  - Time-of-Use (TOU) rates (if available)
- Normalizes values to USD, USD/kWh, and kWh buckets

### 🖥️ FastAPI JSON API
Clean REST endpoints:

```
/rates/{utility}/{plan}
```

Example:

```
GET /rates/cemc/residential
```

Returns:

```json
{
  "utility": "CEMC",
  "rates": {
    "residential_standard": {
      "customer_charge_monthly_usd": 39.0,
      "energy_rate_usd_per_kwh": 0.08058,
      "tva_fuel_rate_usd_per_kwh": 0.02177,
      "total_rate_usd_per_kwh": 0.10235
    }
  }
}
```

### ⚡ Home Assistant Integration
Use it directly in HA via the custom integration:

- Automatically registers a cost sensor with the Energy Dashboard.
- Provides:
  - Energy rate  
  - Fuel rate  
  - Total rate  
  - Fixed monthly charge  

### ☸️ Kubernetes-Ready Deployment
Includes:

- Docker/Buildah Containerfile  
- Helm chart  
- Gateway API HTTPRoute  
- TLS via cert-manager  
- Longhorn/NFS persistent cache  
- CronJob for nightly rate refresh  

### 🛠️ CI/CD Enabled
- GitHub Actions builds and publishes container images to GHCR
- Linting, tests, and packaging checks
- Automated Helm chart packaging

---

## 📦 Installation

### 🐳 Run Locally (Development)
```bash
pip install -e .
uvicorn eratemanager.api:app --reload --port 8000
```

### 🐳 Run in Docker
```bash
buildah bud -t ghcr.io/<youruser>/eratemanager .
buildah push ghcr.io/<youruser>/eratemanager
```

### ☸️ Deploy via Helm
```bash
helm upgrade --install eratemanager ./helm/eratemanager   --set image.repository=ghcr.io/<youruser>/eratemanager   --set gatewayAPI.hostname="rates.example.com"
```

---

## 🧩 Home Assistant Integration

A companion HA integration is available:

- Select your provider (CEMC today—more coming)
- Enter your API endpoint URL
- Automatically registers your rate in the Energy Dashboard
- Supports total blended rate from the parser

Example config:

```
sensor.cemc_total_rate
```

---

## 📂 Project Structure

```
eratemanager/
├── eratemanager/          # Python package
│   ├── parser.py          # PDF parsing logic
│   ├── normalizer.py      # Rate normalization
│   ├── api.py             # FastAPI app
│   ├── downloader.py      # PDF downloader + cache
│   └── cli.py             # CLI tool
├── helm/eratemanager      # Helm chart
├── tests/                 # Pytest suite
├── pyproject.toml         # Packaging config
├── Containerfile          # Container build
├── Makefile               # Build automation
└── README.md
```

---

## 🧪 Testing

```bash
pytest -v
```

Tests include:

- PDF snapshots  
- Parser extraction accuracy  
- Normalization correctness  
- API behavior  
- Version drift detection  

---

## 🛣️ Roadmap

- Add more Tennessee utilities (MLGW, NES, TVA distributors)
- Build a Utility Rate Provider Registry
- UI dashboard for visualizing rate changes
- Publish to PyPI
- Expand HA integration to:
  - TOU window sensors
  - Seasonal sensors  
  - Automatically choose the correct plan

---

## 🤝 Contributing

Pull requests are welcome!  
For major changes, open an issue first to discuss the idea.

---

## 📄 License

MIT License — see LICENSE for details.
