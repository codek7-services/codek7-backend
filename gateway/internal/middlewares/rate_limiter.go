package middlewares

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func RateLimitMiddleware(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      ctx := context.Background()
			ip := r.RemoteAddr
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = strings.Split(fwd, ",")[0]
			}

			key := "rate:" + ip

			// Increment request count
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Set expiration on first request
			if count == 1 {
				rdb.Expire(ctx, key, window)
			}

			if count > int64(limit) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

