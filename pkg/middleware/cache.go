package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/types"
)

type CacheKey = types.CacheKey

func Cache(cache *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var ctx context.Context

			var cacheMap = make(map[string]interface{})
			cacheMap["cache"] = cache

			// Determine the day used for cache keys. If a `date` query param is
			// provided and parses as YYYY-MM-DD, use that; otherwise use today.
			day := time.Now().Format("2006-01-02")
			if d := r.URL.Query().Get("date"); d != "" {
				if parsed, err := time.Parse("2006-01-02", d); err == nil {
					day = parsed.Format("2006-01-02")
				}
			}

			response, found := cache.Get(fmt.Sprint(r.URL) + day)
			if found {
				w.Header().Set("Cache-Status", "HIT")
				cacheMap["response"] = response

			} else {
				w.Header().Set("Cache-Status", "MISS")

			}

			ctx = context.WithValue(r.Context(), CacheKey{}, cacheMap)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
