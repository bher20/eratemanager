package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"context"

	"github.com/bher20/eratemanager/internal/api/swagger"
	"github.com/bher20/eratemanager/internal/auth"
	"github.com/bher20/eratemanager/internal/metrics"
	migrate "github.com/bher20/eratemanager/internal/migrate"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
	"github.com/bher20/eratemanager/internal/ui"
	"github.com/robfig/cron/v3"
)

// NewMux constructs the HTTP mux, wiring in the rates service, metrics, and health endpoints.
func NewMux() *http.ServeMux {
	// Build PDF paths map from environment variables and provider defaults
	pdfPaths := make(map[string]string)
	for _, p := range rates.Providers() {
		// Check for env var override first (e.g., CEMC_PDF_PATH, NES_PDF_PATH)
		envKey := strings.ToUpper(p.Key) + "_PDF_PATH"
		if path := os.Getenv(envKey); path != "" {
			pdfPaths[p.Key] = path
		} else if p.DefaultPDFPath != "" {
			pdfPaths[p.Key] = p.DefaultPDFPath
		}
	}
	cfg := rates.Config{PDFPaths: pdfPaths}

	// Optional auto-migration: run `goose up` on startup when enabled.
	autoMig := os.Getenv("ERATEMANAGER_AUTO_MIGRATE")
	driver := os.Getenv("ERATEMANAGER_DB_DRIVER")
	dsn := os.Getenv("ERATEMANAGER_DB_DSN")
	if autoMig == "1" || strings.ToLower(autoMig) == "true" || strings.ToLower(autoMig) == "yes" {
		ctx := context.Background()
		if driver == "" {
			driver = "sqlite"
		}
		if dsn == "" {
			dsn = "eratemanager.db"
		}
		if err := migrate.Up(ctx, driver, dsn); err != nil {
			log.Printf("auto-migration failed: %v", err)
		}
	}

	// Construct the rates service, preferring a real storage backend when available.
	var svc *rates.Service
	ctxSvc := context.Background()
	// When using the in-memory storage, preload provider descriptors so the
	// UI and cron workers know which providers exist without calling into
	// storage (avoids import cycles).
	var st storage.Storage
	var err error
	if driver == "memory" {
		// Convert rates.ProviderDescriptor -> storage.Provider
		var pList []storage.Provider
		for _, pd := range rates.Providers() {
			pList = append(pList, storage.Provider{
				Key:            pd.Key,
				Name:           pd.Name,
				LandingURL:     pd.LandingURL,
				DefaultPDFPath: pd.DefaultPDFPath,
				Notes:          pd.Notes,
			})
		}
		st = storage.NewMemoryWithProviders(pList)
		err = nil
	} else {
		st, err = storage.Open(ctxSvc, storage.Config{Driver: driver, DSN: dsn})
	}
	if err != nil {
		log.Printf("storage.Open failed (driver=%s dsn=%s): %v; falling back to PDF-only mode", driver, dsn, err)
		svc = rates.NewService(cfg)
	} else {
		log.Printf("rates service using storage backend driver=%s", driver)
		svc = rates.NewServiceWithStorage(cfg, st)
	}

	// Initialize Auth Service
	var authSvc *auth.Service
	if st != nil {
		authSvc, err = auth.NewService(st)
		if err != nil {
			log.Printf("failed to initialize auth service: %v", err)
		} else {
			// Check if users exist, but do NOT create default admin automatically
			// This allows the UI to detect uninitialized state and prompt for setup
			ctx := context.Background()
			users, err := st.ListUsers(ctx)
			if err == nil && len(users) == 0 {
				log.Println("No users found. Waiting for initial setup via UI.")
			}
		}
	}

	mux := http.NewServeMux()

	if authSvc != nil {
		mux.HandleFunc("/auth/status", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			users, err := st.ListUsers(r.Context())
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{
				"initialized": len(users) > 0,
			})
		})

		mux.HandleFunc("/auth/setup", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			// Check if already initialized
			users, err := st.ListUsers(r.Context())
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if len(users) > 0 {
				http.Error(w, "System already initialized", http.StatusForbidden)
				return
			}

			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			if req.Username == "" || req.Password == "" {
				http.Error(w, "Username and password required", http.StatusBadRequest)
				return
			}

			user, err := authSvc.Register(r.Context(), req.Username, req.Password, "admin")
			if err != nil {
				log.Printf("Failed to create user: %v", err)
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		})

		mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			user, err := authSvc.Authenticate(r.Context(), req.Username, req.Password)
			if err != nil {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			// Clean up expired session tokens for this user to prevent accumulation
			if existingTokens, err := st.ListTokens(r.Context(), user.ID); err == nil {
				now := time.Now()
				for _, token := range existingTokens {
					if token.Name == "session" && token.ExpiresAt != nil && token.ExpiresAt.Before(now) {
						st.DeleteToken(r.Context(), token.ID)
					}
				}
			}

			expiresAt := time.Now().Add(24 * time.Hour)
			_, tokenValue, err := authSvc.CreateToken(r.Context(), user.ID, "session", user.Role, &expiresAt)
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"token": tokenValue,
				"user":  user,
			})
		})

		// Token management endpoints
		mux.Handle("/auth/tokens", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				tokenObj, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
				if !ok {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				tokens, err := st.ListTokens(r.Context(), tokenObj.UserID)
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tokens)
				return
			}
			if r.Method == http.MethodPost {
				var req struct {
					Name      string `json:"name"`
					Role      string `json:"role"`
					ExpiresIn string `json:"expires_in"` // e.g., "30d", "never", "24h"
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}

				tokenObj, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
				if !ok {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				// Parse expiration duration (supports both relative durations and custom dates)
				expiresAt, err := auth.ParseExpirationDuration(req.ExpiresIn)
				if err != nil {
					http.Error(w, fmt.Sprintf("Invalid expires_in: %v", err), http.StatusBadRequest)
					return
				}

				t, val, err := authSvc.CreateToken(r.Context(), tokenObj.UserID, req.Name, req.Role, expiresAt)
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"token":       t,
					"token_value": val,
				})
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})))

		mux.Handle("/auth/tokens/", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			id := strings.TrimPrefix(r.URL.Path, "/auth/tokens/")
			if id == "" {
				http.Error(w, "Missing ID", http.StatusBadRequest)
				return
			}

			tokenObj, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			target, err := st.GetToken(r.Context(), id)
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			if target.UserID != tokenObj.UserID {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			if err := st.DeleteToken(r.Context(), id); err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		})))

		// Users management
		mux.Handle("/auth/users", authSvc.Middleware(authSvc.RequirePermission("users", "read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			users, err := st.ListUsers(r.Context())
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			// Redact password hashes
			for i := range users {
				users[i].PasswordHash = ""
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(users)
		}))))

		// Roles
		mux.Handle("/auth/roles", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(auth.RoleContextKey).(string)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if r.Method == http.MethodGet {
				allowed, err := authSvc.Enforce(role, "roles", "read")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				roles, err := authSvc.GetAllRoles()
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(roles)
				return
			}

			if r.Method == http.MethodPost {
				allowed, err := authSvc.Enforce(role, "roles", "write")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				var req struct {
					Role     string        `json:"role"`
					Policies []auth.Policy `json:"policies"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}
				if req.Role == "" {
					http.Error(w, "Role name required", http.StatusBadRequest)
					return
				}
				if _, err := authSvc.CreateRole(req.Role, req.Policies); err != nil {
					http.Error(w, "Failed to create role", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]bool{"success": true})
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})))

		// Privileges (Policies)
		mux.Handle("/auth/privileges", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(auth.RoleContextKey).(string)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if r.Method == http.MethodGet {
				allowed, err := authSvc.Enforce(role, "privileges", "read")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				rawPolicies, err := authSvc.GetAllPolicies()
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}

				type Policy struct {
					Role     string `json:"role"`
					Resource string `json:"resource"`
					Action   string `json:"action"`
				}

				var policies []Policy
				for _, p := range rawPolicies {
					if len(p) >= 3 {
						policies = append(policies, Policy{
							Role:     p[0],
							Resource: p[1],
							Action:   p[2],
						})
					}
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(policies)
				return
			}

			if r.Method == http.MethodPost {
				allowed, err := authSvc.Enforce(role, "privileges", "write")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				var req struct {
					Role     string `json:"role"`
					Resource string `json:"resource"`
					Action   string `json:"action"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}

				if _, err := authSvc.AddPolicy(req.Role, req.Resource, req.Action); err != nil {
					http.Error(w, "Failed to add policy", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]bool{"success": true})
				return
			}

			if r.Method == http.MethodDelete {
				allowed, err := authSvc.Enforce(role, "privileges", "write")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				var req struct {
					Role     string `json:"role"`
					Resource string `json:"resource"`
					Action   string `json:"action"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}

				if _, err := authSvc.RemovePolicy(req.Role, req.Resource, req.Action); err != nil {
					http.Error(w, "Failed to remove policy", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]bool{"success": true})
				return
			}

			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})))
	}

	// Metrics endpoint.
	mux.Handle("/metrics", promhttp.Handler())

	// Health / readiness / liveness.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		drv := os.Getenv("ERATEMANAGER_DB_DRIVER")
		dsn := os.Getenv("ERATEMANAGER_DB_DSN")
		if drv == "" {
			drv = "sqlite"
		}
		if dsn == "" {
			dsn = "eratemanager.db"
		}
		st, err := storage.Open(ctx, storage.Config{Driver: drv, DSN: dsn})
		if err != nil {
			log.Printf("readyz: storage open failed: %v", err)
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		defer st.Close()
		if err := st.Ping(ctx); err != nil {
			log.Printf("readyz: db ping failed: %v", err)
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("live"))
	})

	// Rates API - Electric (includes refresh endpoint)
	electricHandler := http.Handler(handleElectricRates(svc, st, authSvc))
	if authSvc != nil {
		electricHandler = authSvc.Middleware(authSvc.RequirePermission("rates", "read", electricHandler))
	}
	mux.Handle("/rates/electric/", electricHandler)

	RegisterProvidersHandler(mux, authSvc)

	// Rates API - Water
	RegisterWaterHandlers(mux, st, authSvc)

	// System Info
	mux.HandleFunc("/system/info", func(w http.ResponseWriter, r *http.Request) {
		drv := os.Getenv("ERATEMANAGER_DB_DRIVER")
		if drv == "" {
			drv = "sqlite"
		}

		// Format for display
		displayStorage := "SQLite"
		if drv == "postgres" {
			displayStorage = "PostgreSQL"
		} else if drv != "sqlite" {
			displayStorage = drv
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"storage": displayStorage,
		})
	})

	// Settings API
	mux.HandleFunc("/settings/refresh-interval", handleRefreshInterval(st))

	// Swagger API documentation
	mux.Handle("/swagger/", http.StripPrefix("/swagger", swagger.Handler()))

	// Web UI
	mux.Handle("/ui/", http.StripPrefix("/ui/", ui.Handler()))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	return mux
}

// handleElectricRates returns a handler that serves /rates/electric/{provider}/residential, /rates/electric/{provider}/pdf, and /rates/electric/{provider}/refresh.
func handleElectricRates(svc *rates.Service, st storage.Storage, authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Expected paths: /rates/electric/{provider}/residential, /rates/electric/{provider}/pdf, or /rates/electric/{provider}/refresh
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) != 4 || parts[0] != "rates" || parts[1] != "electric" {
			metrics.RequestErrorsTotal.WithLabelValues("unknown", r.URL.Path, "404").Inc()
			http.NotFound(w, r)
			return
		}

		providerKey := strings.ToLower(parts[2])
		endpoint := parts[3]

		// Handle refresh
		if endpoint == "refresh" {
			if authSvc != nil {
				token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
				if !ok {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				allowed, err := authSvc.Enforce(token.UserID, "rates", "write")
				if err != nil {
					http.Error(w, "Internal Error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}
			handleElectricRefresh(w, r, providerKey, st, start)
			return
		}

		// Handle PDF download
		if endpoint == "pdf" {
			labelsPath := "/rates/electric/pdf"
			defer func() {
				dur := time.Since(start).Seconds()
				metrics.RequestDurationSeconds.WithLabelValues(providerKey, labelsPath).Observe(dur)
			}()
			metrics.RequestsTotal.WithLabelValues(providerKey).Inc()

			p, ok := rates.GetProvider(providerKey)
			if !ok {
				metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
				http.NotFound(w, r)
				return
			}

			pdfPath := p.DefaultPDFPath
			if pdfPath == "" {
				metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
				http.Error(w, "no PDF configured for this provider", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_rates.pdf", providerKey))
			http.ServeFile(w, r, pdfPath)
			return
		}

		// Handle residential rates
		if endpoint != "residential" {
			metrics.RequestErrorsTotal.WithLabelValues("unknown", r.URL.Path, "404").Inc()
			http.NotFound(w, r)
			return
		}

		labelsProvider := providerKey
		labelsPath := "/rates/residential"

		defer func() {
			dur := time.Since(start).Seconds()
			metrics.RequestDurationSeconds.WithLabelValues(labelsProvider, labelsPath).Observe(dur)
		}()

		metrics.RequestsTotal.WithLabelValues(labelsProvider).Inc()

		resp, err := svc.GetResidential(r.Context(), providerKey)
		if err != nil {
			log.Printf("get residential rates for %s failed: %v", providerKey, err)
			errMsg := err.Error()

			// Check for specific error types and return appropriate status codes
			if strings.Contains(errMsg, "PDF not found") || strings.Contains(errMsg, "no such file") {
				metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "404").Inc()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "rates_not_available",
					"message": fmt.Sprintf("Rate data for %s is not yet available. Please click 'Refresh Rates' to fetch the latest data.", providerKey),
				})
				return
			}
			if strings.Contains(errMsg, "unknown provider") || strings.Contains(errMsg, "no parser registered") {
				metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "404").Inc()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "provider_not_found",
					"message": fmt.Sprintf("Provider '%s' is not configured.", providerKey),
				})
				return
			}

			metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "500").Inc()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "internal_error",
				"message": "An unexpected error occurred while fetching rates.",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("encode response failed: %v", err)
			metrics.RequestErrorsTotal.WithLabelValues(labelsProvider, labelsPath, "500").Inc()
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

// handleElectricRefresh handles the refresh endpoint for electric providers.
func handleElectricRefresh(w http.ResponseWriter, r *http.Request, providerKey string, st storage.Storage, start time.Time) {
	labelsPath := "/rates/electric/refresh"
	defer func() {
		dur := time.Since(start).Seconds()
		metrics.RequestDurationSeconds.WithLabelValues(providerKey, labelsPath).Observe(dur)
	}()

	metrics.RequestsTotal.WithLabelValues(providerKey).Inc()

	p, ok := rates.GetProvider(providerKey)
	if !ok {
		log.Printf("unknown provider %q for refresh", providerKey)
		metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "404").Inc()
		http.NotFound(w, r)
		return
	}

	// Step 1: Download the latest PDF
	pdfURL, err := rates.RefreshProviderPDF(p)

	resp := RefreshResponse{
		Provider: providerKey,
		PDFURL:   pdfURL,
		Path:     p.DefaultPDFPath,
	}

	if err != nil {
		log.Printf("refresh PDF for %s failed: %v", providerKey, err)
		resp.Status = "error"
		resp.Error = err.Error()
		metrics.RequestErrorsTotal.WithLabelValues(providerKey, labelsPath, "500").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Step 2: Parse the PDF to extract rates
	ratesResp, parseErr := rates.ParseProviderPDF(providerKey)
	if parseErr != nil {
		log.Printf("parse PDF for %s failed: %v", providerKey, parseErr)
		resp.Status = "partial"
		resp.Error = "PDF downloaded but parsing failed: " + parseErr.Error()
	} else {
		resp.Status = "success"
		resp.Rates = ratesResp

		// Save to storage if available
		if st != nil && ratesResp != nil {
			payload, _ := json.Marshal(ratesResp)
			_ = st.SaveRatesSnapshot(r.Context(), storage.RatesSnapshot{
				Provider:  providerKey,
				Payload:   payload,
				FetchedAt: ratesResp.FetchedAt,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleRefreshInterval(st storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method == http.MethodGet {
			val, err := st.GetSetting(ctx, "refresh_interval_seconds")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if val == "" {
				val = "3600" // Default to 1 hour
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"interval": val})
			return
		}
		if r.Method == http.MethodPost {
			var req struct {
				Interval string `json:"interval"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Validate integer or cron expression
			if _, err := strconv.Atoi(req.Interval); err != nil {
				// Not an integer, check if it's a valid cron expression
				if _, cronErr := cron.ParseStandard(req.Interval); cronErr != nil {
					http.Error(w, "invalid interval or cron expression", http.StatusBadRequest)
					return
				}
			}
			if err := st.SetSetting(ctx, "refresh_interval_seconds", req.Interval); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
