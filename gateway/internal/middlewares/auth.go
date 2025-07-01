package middlewares

import (
	"fmt"
	"net/http"

	"github.com/lumbrjx/codek7/gateway/pkg/utils"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the session_token cookie
		fmt.Println("AuthMiddleware: Checking session_token cookie")
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized: Missing session token", http.StatusUnauthorized)
			fmt.Println("AuthMiddleware: Missing or invalid session_token cookie", err)
			return

		}
		fmt.Println("AuthMiddleware: Found session_token cookie:", cookie.Value);

		// Validate JWT token
		userID, err := utils.ValidateToken(cookie.Value)
		fmt.Println("AuthMiddleware: Validating userId:", userID)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			fmt.Println("AuthMiddleware: Invalid token:", err)
			return
		}

		fmt.Println("AuthMiddleware: Validated user ID:", userID)

		// Set user ID in context for use in handlers
		ctx := utils.WithUserID(r.Context(), userID)
		// mark that it passed by a middleware
		ctx = utils.WithAuthPassed(ctx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
