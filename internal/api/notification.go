package api

import (
	"encoding/json"
	"net/http"

	"github.com/bher20/eratemanager/internal/auth"
	"github.com/bher20/eratemanager/internal/notification"
	"github.com/bher20/eratemanager/internal/storage"
)

func registerNotificationRoutes(mux *http.ServeMux, authSvc *auth.Service, notifSvc *notification.Service) {
	mux.Handle("/api/v1/settings/email", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == http.MethodGet {
			allowed, err := authSvc.Enforce(token.UserID, "settings", "read")
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			cfg, err := notifSvc.GetConfig(r.Context())
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			if cfg == nil {
				cfg = &storage.EmailConfig{}
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cfg)
			return
		}

		if r.Method == http.MethodPut {
			allowed, err := authSvc.Enforce(token.UserID, "settings", "write")
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			var req storage.EmailConfig
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			if err := notifSvc.SaveConfig(r.Context(), req); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})))

	mux.Handle("/api/v1/settings/email/test", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		allowed, err := authSvc.Enforce(token.UserID, "settings", "write")
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		var req struct {
			Config storage.EmailConfig `json:"config"`
			To     string              `json:"to"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := notifSvc.TestConfig(r.Context(), req.Config, req.To); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	})))
}
