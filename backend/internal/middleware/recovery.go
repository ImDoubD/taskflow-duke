package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

// Recovery catches panics in handlers, logs them, and returns 500.
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("panic recovered", "panic", rec, "path", r.URL.Path)
					writeJSONError(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// contextWith is a typed helper to avoid allocating interface{} keys inline.
func contextWith(ctx context.Context, key contextKey, val string) context.Context {
	return context.WithValue(ctx, key, val)
}
