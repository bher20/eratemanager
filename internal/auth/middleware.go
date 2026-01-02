package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/bher20/eratemanager/internal/storage"
)

type contextKey string

const (
	UserContextKey  contextKey = "user"
	TokenContextKey contextKey = "token"
	RoleContextKey  contextKey = "role"
)

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		tokenValue := parts[1]
		token, err := s.ValidateToken(r.Context(), tokenValue)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), TokenContextKey, token)
		ctx = context.WithValue(ctx, RoleContextKey, token.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) RequirePermission(obj, act string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := r.Context().Value(TokenContextKey).(*storage.Token)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		allowed, err := s.Enforce(token.UserID, obj, act)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
