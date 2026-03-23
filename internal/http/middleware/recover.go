package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"request_id", GetRequestID(r.Context()),
						"panic", rec,
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"code":    "internal_error",
						"message": "internal server error",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
