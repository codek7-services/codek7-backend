package middlewares

import (
	"net/http"

	"github.com/lumbrjx/codek7/gateway/pkg/utils"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the session_token cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized: Missing session token", http.StatusUnauthorized)
			return
		}

		// Validate JWT token
		userID, err := utils.ValidateToken(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Set user ID in context for use in handlers
		ctx := utils.WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
