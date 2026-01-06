package api

import (
"encoding/json"
"net/http"
"os"
"strings"

"github.com/bher20/eratemanager/internal/auth"
"github.com/bher20/eratemanager/internal/rates"
"github.com/bher20/eratemanager/internal/storage"
"github.com/bher20/eratemanager/pkg/providers/electricproviders"
"github.com/bher20/eratemanager/pkg/providers/waterproviders"
)

// ProviderDTO represents a provider in the API.
type ProviderDTO struct {
Key  string `json:"key"`
Name string `json:"name"`
Type string `json:"type"`
}

type V2Handler struct {
svc      *rates.Service
waterSvc *rates.WaterService
st       storage.Storage
authSvc  *auth.Service
}

func RegisterV2Routes(mux *http.ServeMux, svc *rates.Service, waterSvc *rates.WaterService, st storage.Storage, authSvc *auth.Service) {
h := &V2Handler{
svc:      svc,
waterSvc: waterSvc,
st:       st,
authSvc:  authSvc,
}

// Helper to wrap handler with auth middleware if authSvc is present
withAuth := func(handler http.HandlerFunc) http.Handler {
if authSvc == nil {
return handler
}
return authSvc.Middleware(handler)
}

mux.Handle("/api/v2/electric-rates/providers", withAuth(h.ListElectricProviders))
mux.Handle("/api/v2/electric-rates/", withAuth(h.HandleElectricRates))
mux.Handle("/api/v2/water-rates/providers", withAuth(h.ListWaterProviders))
mux.Handle("/api/v2/water-rates/", withAuth(h.HandleWaterRates))
}

// ListElectricProviders lists all electric providers
// @Summary List electric providers
// @Description Get a list of all available electric providers
// @Tags electric
// @Produce json
// @Success 200 {array} ProviderDTO
// @Router /api/v2/electric-rates/providers [get]
func (h *V2Handler) ListElectricProviders(w http.ResponseWriter, r *http.Request) {
if h.authSvc != nil {
if allowed, err := h.authSvc.Enforce(getUserID(r), "providers", "read"); err != nil || !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
if r.Method != http.MethodGet {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}
providers := electricproviders.GetAll()
var list []ProviderDTO
for _, p := range providers {
list = append(list, ProviderDTO{
Key:  p.Key(),
Name: p.Name(),
Type: string(p.Type()),
})
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(list)
}

// HandleElectricRates handles electric rates requests
// @Summary Get electric rates or refresh
// @Description Get residential rates, PDF, or force refresh for a provider
// @Tags electric
// @Produce json
// @Param providerKey path string true "Provider Key"
// @Param action path string true "Action (residential, pdf, refresh)"
// @Router /api/v2/electric-rates/{providerKey}/{action} [get]
func (h *V2Handler) HandleElectricRates(w http.ResponseWriter, r *http.Request) {
path := strings.TrimPrefix(r.URL.Path, "/api/v2/electric-rates/")
parts := strings.Split(path, "/")
if len(parts) < 2 {
http.NotFound(w, r)
return
}
providerKey := parts[0]
endpoint := parts[1]

if endpoint == "refresh" {
if r.Method != http.MethodPost {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}
if h.authSvc != nil {
if allowed, err := h.authSvc.Enforce(getUserID(r), "rates", "write"); err != nil || !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
resp, err := h.svc.ForceRefresh(r.Context(), providerKey)
if err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(resp)
return
}

if endpoint == "residential" {
if r.Method != http.MethodGet {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}
if h.authSvc != nil {
if allowed, err := h.authSvc.Enforce(getUserID(r), "rates", "read"); err != nil || !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
resp, err := h.svc.GetElectricResidential(r.Context(), providerKey)
if err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(resp)
return
}

if endpoint == "pdf" {
if r.Method != http.MethodGet {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}
if h.authSvc != nil {
if allowed, err := h.authSvc.Enforce(getUserID(r), "rates", "read"); err != nil || !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
// Legacy way to get path
p, ok := electricproviders.Get(providerKey)
if !ok || p.DefaultPDFPath() == "" {
http.NotFound(w, r)
return
}
// Check if file exists
if _, err := os.Stat(p.DefaultPDFPath()); err != nil {
http.NotFound(w, r)
return
}
http.ServeFile(w, r, p.DefaultPDFPath())
return
}

http.NotFound(w, r)
}

// ListWaterProviders lists all water providers
// @Summary List water providers
// @Description Get a list of all available water providers
// @Tags water
// @Produce json
// @Success 200 {array} ProviderDTO
// @Router /api/v2/water-rates/providers [get]
func (h *V2Handler) ListWaterProviders(w http.ResponseWriter, r *http.Request) {
if h.authSvc != nil {
if allowed, err := h.authSvc.Enforce(getUserID(r), "providers", "read"); err != nil || !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
if r.Method != http.MethodGet {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}
providers := waterproviders.GetAll()
var list []ProviderDTO
for _, p := range providers {
list = append(list, ProviderDTO{
Key:  p.Key(),
Name: p.Name(),
Type: string(p.Type()),
})
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(list)
}

// HandleWaterRates handles water rates requests
// @Summary Get water rates or refresh
// @Description Get water rates or force refresh for a provider
// @Tags water
// @Produce json
// @Param providerKey path string true "Provider Key"
// @Param action path string false "Action (refresh)"
// @Router /api/v2/water-rates/{providerKey} [get]
func (h *V2Handler) HandleWaterRates(w http.ResponseWriter, r *http.Request) {
path := strings.TrimPrefix(r.URL.Path, "/api/v2/water-rates/")
parts := strings.Split(path, "/")
if len(parts) < 1 {
http.NotFound(w, r)
return
}
providerKey := parts[0]

if len(parts) > 1 && parts[1] == "refresh" {
if r.Method != http.MethodPost {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}
if h.authSvc != nil {
if allowed, err := h.authSvc.Enforce(getUserID(r), "rates", "write"); err != nil || !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
resp, err := h.waterSvc.ForceRefresh(r.Context(), providerKey)
if err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(resp)
return
}

// Get Rates
if r.Method != http.MethodGet {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}
if h.authSvc != nil {
if allowed, err := h.authSvc.Enforce(getUserID(r), "rates", "read"); err != nil || !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
resp, err := h.waterSvc.GetWaterRates(r.Context(), providerKey)
if err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}
if resp == nil {
http.NotFound(w, r)
return
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(resp)
}

func getUserID(r *http.Request) string {
token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
if !ok {
return ""
}
return token.UserID
}
