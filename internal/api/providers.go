package api

import (
	"encoding/json"
	"net/http"

	"github.com/bher20/eratemanager/internal/rates"
)

func RegisterProvidersHandler(mux *http.ServeMux) {
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		providers := rates.Providers()

		response := struct {
			Providers []rates.ProviderDescriptor `json:"providers"`
		}{
			Providers: providers,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}
