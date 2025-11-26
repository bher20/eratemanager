# Adding a New Utility Provider to eRateManager (Go)

This project uses a **parser registry pattern** that makes adding new providers
simple. Parsers auto-register themselves via Go's `init()` mechanism.

## Quick Start: 2 Steps

### Step 1: Run the scaffold script

```bash
python tools/scaffold_provider.py \
    --key myutility \
    --name "My Utility Power" \
    --landing-url "https://myutility.example.com/rates/" \
    --pdf-path "/data/myutility_rates.pdf"
```

This generates:
- `internal/rates/parser_myutility_pdf.go` - Parser with auto-registration
- `internal/rates/parser_myutility_pdf_test.go` - Unit test template

### Step 2: Add provider descriptor

Add an entry to `internal/rates/providers.go`:

```go
{
    Key:            "myutility",
    Name:           "My Utility Power",
    LandingURL:     "https://myutility.example.com/rates/",
    DefaultPDFPath: "/data/myutility_rates.pdf",
    Notes:          "My Utility residential rates.",
},
```

**That's it!** The parser auto-registers via `init()`. No switch statements
to update, no service.go changes needed.

### Step 3 (Optional): Customize the parser

Edit `internal/rates/parser_myutility_pdf.go` to match your PDF's actual format:

```go
// Customize these regex patterns to match your PDF
custRe := regexp.MustCompile(`Customer Charge[:\s]*\$?([0-9]+(?:\.[0-9]+)?)`)
energyRe := regexp.MustCompile(`Energy Charge[:\s]*([0-9]+(?:\.[0-9]+)?)\s*cents?\s*per kWh`)
```

Then verify:

```bash
gofmt -w internal/rates/parser_myutility_pdf.go
go test ./...
```

## How It Works

Each parser file contains an `init()` function that registers itself:

```go
func init() {
    RegisterParser(ParserConfig{
        Key:       "myutility",
        Name:      "My Utility Power",
        ParsePDF:  ParseMyutilityRatesFromPDF,
        ParseText: ParseMyutilityRatesFromText,
    })
}
```

The service layer uses the registry to find parsers dynamically:

```go
// No switch statements needed - just registry lookup
parser, ok := GetParser(providerKey)
if !ok {
    return nil, fmt.Errorf("unknown provider: %s", providerKey)
}
return parser.ParsePDF(pdfPath)
```

## Optional Integrations

### Helm Chart

Add PDF configuration to `helm/eratemanager/values.yaml`:

```yaml
pdf:
  myutility:
    url: "https://myutility.example.com/path/to/rates.pdf"
    path: "/data/myutility_rates.pdf"
```

### Home Assistant Integration

Add to `ha_energy_rates` integration's `const.py`:

```python
"myutility": {
    "name": "My Utility Power",
    "default_url": "https://<your-hostname>/rates/myutility/residential",
},
```

## Testing

```bash
# Run all tests
go test ./...

# Test the endpoint
curl https://<your-hostname>/rates/myutility/residential
```

## Architecture Overview

```
internal/rates/
├── registry.go          # Parser registration system
├── providers.go         # Provider metadata (name, URLs, paths)
├── service.go           # Rate fetching service (uses registry)
├── model.go             # RatesResponse data structures
├── parser_cemc_pdf.go   # CEMC parser (auto-registers)
├── parser_nes_pdf.go    # NES parser (auto-registers)
└── parser_*.go          # Your new parsers (auto-register)
```
