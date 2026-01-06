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
	migrate "github.com/bher20/eratemanager/internal/migrate"
	"github.com/bher20/eratemanager/internal/notification"
	"github.com/bher20/eratemanager/internal/rates"
	"github.com/bher20/eratemanager/internal/storage"
	"github.com/bher20/eratemanager/internal/version"
	"github.com/bher20/eratemanager/internal/ui"
	"github.com/bher20/eratemanager/pkg/providers/electricproviders"
	"github.com/bher20/eratemanager/pkg/providers/waterproviders"
	"github.com/robfig/cron/v3"
)

// @title eRateManager API
// @version 2.0
// @description API for eRateManager
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// NewMux constructs the HTTP mux, wiring in the rates service, metrics, and health endpoints.
func NewMux() *http.ServeMux {
	// Build PDF paths map from environment variables and provider defaults
	pdfPaths := make(map[string]string)
	for _, p := range electricproviders.GetAll() {
		// Check for env var override first (e.g., CEMC_PDF_PATH, NES_PDF_PATH)
		envKey := strings.ToUpper(p.Key()) + "_PDF_PATH"
		if path := os.Getenv(envKey); path != "" {
			pdfPaths[p.Key()] = path
		} else if p.DefaultPDFPath() != "" {
			pdfPaths[p.Key()] = p.DefaultPDFPath()
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
		// Convert providers -> storage.Provider
		var pList []storage.Provider
		for _, p := range electricproviders.GetAll() {
			pList = append(pList, storage.Provider{
				Key:            p.Key(),
				Name:           p.Name(),
				LandingURL:     p.LandingURL(),
				DefaultPDFPath: p.DefaultPDFPath(),
				// Notes:          p.Notes(), // Notes not yet in interface
			})
		}
		for _, p := range waterproviders.GetAll() {
			pList = append(pList, storage.Provider{
				Key:            p.Key(),
				Name:           p.Name(),
				LandingURL:     p.LandingURL(),
				DefaultPDFPath: p.DefaultPDFPath(),
			})
		}
		st = storage.NewMemoryWithProviders(pList)
		err = nil
	} else {
		// Retry connection for up to 30 seconds to allow database to start
		for i := 0; i < 6; i++ {
			st, err = storage.Open(ctxSvc, storage.Config{Driver: driver, DSN: dsn})
			if err == nil {
				break
			}
			log.Printf("storage.Open failed (attempt %d/6): %v; retrying in 5s...", i+1, err)
			time.Sleep(5 * time.Second)
		}
	}
	if err != nil {
		log.Printf("storage.Open failed (driver=%s dsn=%s): %v; falling back to PDF-only mode", driver, dsn, err)
		// If we explicitly requested a database (other than default sqlite fallback), we should probably fail hard
		// so Kubernetes restarts us, rather than running in a broken state.
		if os.Getenv("ERATEMANAGER_DB_DRIVER") != "" {
			log.Fatal("Failed to connect to configured database")
		}
		svc = rates.NewService(cfg)
	} else {
		log.Printf("rates service using storage backend driver=%s", driver)
		svc = rates.NewServiceWithStorage(cfg, st)
	}

	// Initialize Notification Service
	var notifSvc *notification.Service
	if st != nil {
		notifSvc = notification.NewService(st)
	}

	// Initialize Auth Service
	var authSvc *auth.Service
	if st != nil {
		publicURL := os.Getenv("PUBLIC_URL")
		if publicURL == "" {
			publicURL = "http://localhost:8000"
		}
		authSvc, err = auth.NewService(st, notifSvc, publicURL)
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
				Email    string `json:"email"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			if req.Username == "" || req.Password == "" || req.Email == "" {
				http.Error(w, "Username, password, and email required", http.StatusBadRequest)
				return
			}

			user, err := authSvc.Register(r.Context(), req.Username, req.Password, req.Email, "admin")
			if err != nil {
				log.Printf("Failed to create user: %v", err)
				http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
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

		mux.HandleFunc("/auth/forgot-password", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Email string `json:"email"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			
			if req.Email == "" {
				http.Error(w, "Email required", http.StatusBadRequest)
				return
			}

			if err := authSvc.RequestPasswordReset(r.Context(), req.Email); err != nil {
				// Log error but return success to avoid user enumeration
				log.Printf("Password reset request failed: %v", err)
			}
			
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "If an account with that email exists, a password reset link has been sent."}`))
		})

		mux.HandleFunc("/auth/reset-password", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Token       string `json:"token"`
				NewPassword string `json:"new_password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			
			if req.Token == "" || req.NewPassword == "" {
				http.Error(w, "Token and new password required", http.StatusBadRequest)
				return
			}

			if err := authSvc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
				http.Error(w, "Password reset failed: "+err.Error(), http.StatusBadRequest)
				return
			}
			
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Password successfully reset."}`))
		})

		mux.HandleFunc("/auth/validate-setup-token", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Error(w, "Token required", http.StatusBadRequest)
				return
			}

			user, err := authSvc.ValidateSetupToken(r.Context(), token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"email":      user.Email,
				"role":       user.Role,
				"username":   user.Username,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
			})
		})

		mux.HandleFunc("/auth/setup-account", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Token       string `json:"token"`
				Username    string `json:"username"`
				FirstName   string `json:"first_name"`
				LastName    string `json:"last_name"`
				NewPassword string `json:"new_password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			
			if req.Token == "" || req.Username == "" || req.NewPassword == "" {
				http.Error(w, "Token, username and password required", http.StatusBadRequest)
				return
			}

			if err := authSvc.SetupInvitedAccount(r.Context(), req.Token, req.Username, req.FirstName, req.LastName, req.NewPassword); err != nil {
				http.Error(w, "Account setup failed: "+err.Error(), http.StatusBadRequest)
				return
			}
			
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Account successfully set up."}`))
		})


		mux.HandleFunc("/auth/verify-email", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			
			if req.Token == "" {
				http.Error(w, "Token required", http.StatusBadRequest)
				return
			}

			if err := authSvc.VerifyEmail(r.Context(), req.Token); err != nil {
				http.Error(w, "Verification failed: "+err.Error(), http.StatusBadRequest)
				return
			}
			
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Email successfully verified."}`))
		})

		mux.Handle("/auth/resend-verification", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := st.GetUser(r.Context(), token.UserID)
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			if user == nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			if user.EmailVerified {
				http.Error(w, "Email already verified", http.StatusBadRequest)
				return
			}

			if err := authSvc.SendVerificationEmail(r.Context(), user.ID, user.Email); err != nil {
				http.Error(w, "Failed to send email: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Verification email sent."}`))
		})))

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
		mux.Handle("/auth/users", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if r.Method == http.MethodGet {
				allowed, err := authSvc.Enforce(token.UserID, "users", "read")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
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
				return
			}

			if r.Method == http.MethodPost {
				allowed, err := authSvc.Enforce(token.UserID, "users", "write")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				var req struct {
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Password  string `json:"password"`
				Email     string `json:"email"`
				Role      string `json:"role"`
				Invite    bool   `json:"invite"` // If true, send invitation email instead of password
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			var user *storage.User
			if req.Invite || req.Password == "" {
				// Use invitation flow (random password + invitation email)
				user, err = authSvc.RegisterInvitedUser(r.Context(), req.Username, req.FirstName, req.LastName, req.Email, req.Role)
					// Use standard registration flow
					user, err = authSvc.Register(r.Context(), req.Username, req.Password, req.Email, req.Role)
				}
				
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				user.PasswordHash = ""
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(user)
				return
			}

			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})))

		mux.Handle("/auth/users/", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := strings.TrimPrefix(r.URL.Path, "/auth/users/")
			if id == "" {
				http.Error(w, "Missing ID", http.StatusBadRequest)
				return
			}

			token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if r.Method == http.MethodPut {
				allowed, err := authSvc.Enforce(token.UserID, "users", "write")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				var req struct {
					Role  string `json:"role"`
					Email string `json:"email"`
					SkipEmailVerification *bool `json:"skip_email_verification"`
					OnboardingCompleted *bool `json:"onboarding_completed"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request body", http.StatusBadRequest)
					return
				}

				user, err := authSvc.UpdateUser(r.Context(), id, req.Email, req.Role, req.SkipEmailVerification, req.OnboardingCompleted)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				user.PasswordHash = ""
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(user)
				return
			}

			if r.Method == http.MethodDelete {
				allowed, err := authSvc.Enforce(token.UserID, "users", "write")
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				if err := st.DeleteUser(r.Context(), id); err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}

			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})))

		mux.Handle("/auth/users/{id}/resend-invitation", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			allowed, err := authSvc.Enforce(token.UserID, "users", "write")
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Extract user ID from path
			pathParts := strings.Split(r.URL.Path, "/")
			if len(pathParts) < 4 {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			userID := pathParts[3]

			// Get user details
			user, err := st.GetUser(r.Context(), userID)
			if err != nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			// Only resend if user hasn't completed onboarding
			if user.OnboardingCompleted {
				http.Error(w, "User has already completed onboarding", http.StatusBadRequest)
				return
			}

			// Send invitation email
			if err := authSvc.SendInvitationEmail(r.Context(), user.ID, user.Email, user.Role); err != nil {
				http.Error(w, "Failed to send invitation: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Invitation resent successfully"}`))
		})))

		mux.Handle("/auth/me", authSvc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := r.Context().Value(auth.TokenContextKey).(*storage.Token)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if r.Method == http.MethodGet {
				user, err := st.GetUser(r.Context(), token.UserID)
				if err != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				if user == nil {
					http.Error(w, "User not found", http.StatusNotFound)
					return
				}
				user.PasswordHash = ""
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(user)
				return
			}

			if r.Method == http.MethodPut {
				var req struct {
					Email string `json:"email"`
					OnboardingCompleted *bool `json:"onboarding_completed"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request body", http.StatusBadRequest)
					return
				}

				// Users can only update their own email, not role
				user, err := authSvc.UpdateUser(r.Context(), token.UserID, req.Email, "", nil, req.OnboardingCompleted)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				user.PasswordHash = ""
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(user)
				return
			}

			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})))

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

		// Notification Routes
		if notifSvc != nil {
			registerNotificationRoutes(mux, authSvc, notifSvc)
		}
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

	// Initialize Water Service
	var waterSvc *rates.WaterService
	if st != nil {
		waterSvc = rates.NewWaterServiceWithStorage(st)
	} else {
		waterSvc = rates.NewWaterService()
	}

	// Register V2 Routes
	RegisterV2Routes(mux, svc, waterSvc, st, authSvc)

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
			"version": version.Version,
		})
	})

	// Settings API
	mux.HandleFunc("/settings/refresh-interval", handleRefreshInterval(st))
	if notifSvc != nil {
		mux.HandleFunc("/settings/email", handleEmailSettings(notifSvc))
	}

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
				val = "0 0 * * 0" // Default: every Sunday at midnight
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

func handleEmailSettings(svc *notification.Service) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
ctx := r.Context()
if r.Method == http.MethodGet {
cfg, err := svc.GetConfig(ctx)
if err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}
if cfg == nil {
// Return default empty config
cfg = &storage.EmailConfig{
Provider: "smtp",
Port:     587,
Enabled:  true,
}
}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(cfg)
return
}
if r.Method == http.MethodPut {
var cfg storage.EmailConfig
if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
http.Error(w, err.Error(), http.StatusBadRequest)
return
}
if err := svc.SaveConfig(ctx, cfg); err != nil {
http.Error(w, err.Error(), http.StatusInternalServerError)
return
}
w.WriteHeader(http.StatusOK)
return
}
http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
}
