package middleware

import (
	"net/http"
	"strings"

	"github.com/dukedhal/taskflow/internal/service"
)

// Authenticate is a chi middleware that extracts and validates the Bearer JWT.
// On success it injects user_id and email into the request context.
// On failure it returns 401, which is reserved for authorization.
func Authenticate(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSONError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := authSvc.ValidateToken(tokenStr)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := r.Context()
			ctx = contextWith(ctx, contextKeyUserID, claims.UserID)
			ctx = contextWith(ctx, contextKeyEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
