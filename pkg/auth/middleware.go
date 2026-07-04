package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ClaimsKey  contextKey = "claims"
	TokenKey   contextKey = "token"
)

func HTTPMiddleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				http.Error(w, `{"error":"missing authorization token"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtManager.ValidateAccess(token)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			ctx = context.WithValue(ctx, TokenKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func HTTPMiddlewareOptional(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token != "" {
				claims, err := jwtManager.ValidateAccess(token)
				if err == nil {
					ctx := context.WithValue(r.Context(), ClaimsKey, claims)
					ctx = context.WithValue(ctx, TokenKey, token)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}

	cookie, err := r.Cookie("access_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

func GetClaims(ctx context.Context) *Claims {
	claims, _ := ctx.Value(ClaimsKey).(*Claims)
	return claims
}

func GetToken(ctx context.Context) string {
	token, _ := ctx.Value(TokenKey).(string)
	return token
}
