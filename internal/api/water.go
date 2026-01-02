package api

import (
"encoding/json"
"log"
"net/http"
"strings"
"time"

"github.com/bher20/eratemanager/internal/auth"
"github.com/bher20/eratemanager/internal/metrics"
"github.com/bher20/eratemanager/internal/rates"
"github.com/bher20/eratemanager/internal/storage"
)

// RegisterWaterHandlers registers all water-related HTTP handlers.
func RegisterWaterHandlers(mux *http.ServeMux, st storage.Storage, authSvc *auth.Service) {
var waterSvc *rates.WaterService
if st != nil {
waterSvc = rates.NewWaterServiceWithStorage(st)
} else {
waterSvc = rates.NewWaterService()
}

// Water providers list
var providersHandler http.Handler = http.HandlerFunc(handleWaterProviders)
if authSvc != nil {
providersHandler = authSvc.Middleware(authSvc.RequirePermission("providers", "read", providersHandler))
}
mux.Handle("/rates/water/providers", providersHandler)

// Water refresh endpoint (must be registered before rates to match first)
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if strings.HasSuffix(r.URL.Path, "/refresh") {
if authSvc != nil {
role, ok := r.Context().Value(auth.RoleContextKey).(string)
if !ok {
http.Error(w, "Unauthorized", http.StatusUnauthorized)
return
}
allowed, err := authSvc.Enforce(role, "rates", "write")
if err != nil {
http.Error(w, "Internal Error", http.StatusInternalServerError)
return
}
if !allowed {
http.Error(w, "Forbidden", http.StatusForbidden)
return
}
}
handleWaterRefresh(waterSvc)(w, r)
return
}
handleWaterRates(waterSvc)(w, r)
})

var h http.Handler = handler
if authSvc != nil {
h = authSvc.Middleware(authSvc.RequirePermission("rates", "read", h))
}
mux.Handle("/rates/water/", h)
}

// handleWaterProviders returns the list of water providers.
func handleWaterProviders(w http.ResponseWriter, r *http.Request) {
providers := rates.WaterProviders()

w.Header().Set("Content-Type", "application/json")
if err := json.NewEncoder(w).Encode(providers); err != nil {
log.Printf("encode water providers failed: %v", err)
http.Error(w, "internal error", http.StatusInternalServerError)
return
}
}

// handleWaterRates returns a handler for /rates/water/{provider}
func handleWaterRates(svc *rates.WaterService) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
start := time.Now()

// Expected path: /rates/water/{provider}
path := strings.TrimPrefix(r.URL.Path, "/rates/water/")
providerKey := strings.ToLower(strings.TrimSuffix(path, "/"))

if providerKey == "" || providerKey == "providers" {
// Skip if this is the providers endpoint
http.NotFound(w, r)
return
}

labelsPath := "/rates/water"
defer func() {
dur := time.Since(start).Seconds()
metrics.RequestDurationSeconds.WithLabelValues(providerKey, labelsPath).Observe(dur)
}()

metrics.RequestsTotal.WithLabelValues(providerKey).Inc()

resp, err := svc.GetWaterRates(r.Context(), providerKey)
if err != nil {
log.Printf("get water rates for %s failed: %v", providerKey, err)
metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "500").Inc()
http.Error(w, "internal error", http.StatusInternalServerError)
return
}

if resp == nil {
metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
http.NotFound(w, r)
return
}

w.Header().Set("Content-Type", "application/json")
if err := json.NewEncoder(w).Encode(resp); err != nil {
log.Printf("encode water response failed: %v", err)
metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "500").Inc()
http.Error(w, "internal error", http.StatusInternalServerError)
return
}
}
}

// handleWaterRefresh handles /rates/water/{provider}/refresh
func handleWaterRefresh(svc *rates.WaterService) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
// Path: /rates/water/{provider}/refresh
path := strings.TrimPrefix(r.URL.Path, "/rates/water/")
path = strings.TrimSuffix(path, "/refresh")
providerKey := strings.ToLower(strings.Trim(path, "/"))

if providerKey == "" {
http.Error(w, "provider key required", http.StatusBadRequest)
return
}

// Force refresh by fetching directly from source
resp, err := svc.ForceRefresh(r.Context(), providerKey)
if err != nil {
log.Printf("refresh water rates for %s failed: %v", providerKey, err)
http.Error(w, "refresh failed: "+err.Error(), http.StatusInternalServerError)
return
}
if resp == nil {
http.Error(w, "unknown water provider", http.StatusNotFound)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"status":   "refreshed",
"provider": providerKey,
"rates":    resp,
})
}
}
