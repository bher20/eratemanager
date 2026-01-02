package api

import (
	"encoding/json"
	"net/http"

	"github.com/bher20/eratemanager/internal/auth"
	"github.com/bher20/eratemanager/internal/rates"
)

func RegisterProvidersHandler(mux *http.ServeMux, authSvc *auth.Service) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("Providers handler called: %s %s", r.Method, r.URL.Path)
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		providers := rates.Providers()
		// log.Printf("Returning %d providers", len(providers))

		response := struct {
			Providers []rates.ProviderDescriptor `json:"providers"`
		}{
			Providers: providers,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	var h http.Handler = handler
	if authSvc != nil {
		h = authSvc.Middleware(authSvc.RequirePermission("providers", "read", h))
	}
	mux.Handle("/providers", h)
}
