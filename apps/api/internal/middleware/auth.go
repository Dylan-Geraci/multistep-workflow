package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey struct{}

func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				// Fall back to query param for WebSocket connections (browser WS API can't set headers)
				if token := r.URL.Query().Get("token"); token != "" {
					header = "Bearer " + token
				} else {
					model.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
					return
				}
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				model.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
				return
			}

			token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				model.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
				return
			}

			sub, err := token.Claims.GetSubject()
			if err != nil || sub == "" {
				model.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token claims")
				return
			}

			ctx := context.WithValue(r.Context(), contextKey{}, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKey{}).(string); ok {
		return v
	}
	return ""
}

// SetUserIDForTest injects a user ID into context for testing.
func SetUserIDForTest(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKey{}, userID)
}
