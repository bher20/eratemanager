# Adding a New Utility Provider to eRateManager (Go)

This project supports multiple utilities (CEMC, NES, Demo) and provides
scaffolding to add more.

There are three main layers to touch:

1. Go backend (parsing + service)
2. Helm chart (cronjob + storage)
3. Home Assistant integration (provider list + default URL)

## 1. Backend: use the scaffold script

From the repo root, run:

```bash
python tools/scaffold_provider.py       --key myutility       --name "My Utility Power"       --landing-url "https://myutility.example.com/rates/"       --pdf-path "/data/myutility_rates.pdf"
```

This will:

- Generate `internal/rates/parser_myutility_pdf.go` with a stub parser
- Generate `internal/rates/parser_myutility_pdf_test.go` with a sample test
- Print snippets you can copy into:
  - `internal/rates/service.go` (switch case for provider)
  - `internal/rates/providers.go` (descriptor entry)
  - Home Assistant `const.py` (provider entry)
  - Helm `values.yaml` (pdf URL + path)

After generation:

```bash
gofmt -w internal/rates/parser_myutility_pdf.go internal/rates/parser_myutility_pdf_test.go
go test ./...
```

Then implement real regexes in `ParseMyUtilityRatesFromText`.

## 2. Helm chart

- Add a `pdf.<key>.url` entry under `helm/eratemanager/values.yaml`.
- Optionally add a dedicated CronJob template following the existing
  CEMC/NES pattern:
  `helm/eratemanager/templates/cronjob_<key>.yaml`.

## 3. Home Assistant integration

In `ha_energy_rates` (separate repo/integration):

- Add a provider entry to `PROVIDERS` in `const.py`:

  ```python
  "myutility": {
      "name": "My Utility Power",
      "default_url": "https://<your-hostname>/rates/myutility/residential",
  },
  ```

After that, the HA config flow will automatically expose your provider
in the dropdown, and the existing sensors will work as long as your
backend JSON matches the shared schema.

## 4. Testing

- Run backend tests:

  ```bash
  go test ./...
  ```

- Hit the new endpoint once deployed:

  ```bash
  curl https://<your-hostname>/rates/myutility/residential
  ```

- Add the provider in Home Assistant via the "HA Energy Rates" integration.
