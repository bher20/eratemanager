# eRateManager Autodiscovery Update

This bundle contains **backend additions** and an updated **Helm chart** to
support automatic discovery of provider rate PDFs from their landing pages.

## Backend pieces

Files under `backend/internal/rates` and `backend/internal/api` are intended to
be **added to your existing Go backend**:

- `internal/rates/discovery.go`
  - `DiscoverPDFURL(p ProviderDescriptor) (string, error)`
  - `RefreshProviderPDF(p ProviderDescriptor) (string, error)`:
    - Fetches the provider landing page (e.g. CEMC / NES rates page)
    - Discovers the first `href="*.pdf"` link
    - Downloads it
    - Writes it atomically to `ProviderDescriptor.DefaultPDFPath`

- `internal/rates/fileutil.go`
  - `writeFileAtomically(path string, r io.Reader) error`:
    - Safely writes the PDF to disk via a temp file and rename.

- `internal/api/refresh.go`
  - `RegisterRefreshHandler(mux *http.ServeMux)`:
    - Registers `GET /internal/refresh/{provider}`.
    - Calls `rates.RefreshProviderPDF` and returns a small JSON payload.

### Wiring in your existing server

In your existing `internal/api/http.go` (or wherever you construct your mux):

```go
mux := http.NewServeMux()
RegisterRoutes(mux)          // your existing handlers
RegisterRefreshHandler(mux)  // add this line
```

Ensure the import path for `api` matches your module name, e.g.:

```go
import "github.com/bher20/eratemanager/internal/api"
```

and for rates:

```go
import "github.com/bher20/eratemanager/internal/rates"
```

Your existing `providers.go` must expose:

```go
type ProviderDescriptor struct {
    Key            string
    Name           string
    LandingURL     string
    DefaultPDFPath string
    // ...
}

func GetProvider(key string) (ProviderDescriptor, bool)
```

If the actual struct or function names differ, adjust the new files
accordingly.

## Helm chart changes

The Helm chart under `helm/eratemanager` has been updated so that the
CronJobs no longer try to download PDFs directly from the provider URLs.

Instead, they simply **call back into the running eratemanager service**:

```bash
curl -fsSL http://eratemanager:80/internal/refresh/{provider}
```

The backend:
- Discovers the current PDF link
- Downloads it
- Stores it at `/data/{provider}_rates.pdf`
- Logs and updates metrics
- Returns a JSON status

This means future changes to the provider websites (renaming PDF files, etc.)
are handled by the Go autodiscovery code instead of hard-coded URLs in Helm.

## Usage

1. Copy the `backend/internal` files into your existing Go backend.
2. Adjust import paths if your module name is not `github.com/youruser/eratemanager`.
3. Rebuild and push your container image.
4. Deploy the updated Helm chart from `helm/eratemanager`.
5. The CronJobs will now hit `/internal/refresh/{provider}` instead of
   attempting to download from hard-coded `pdfUrl` values.
