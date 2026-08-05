package cors

import (
	"net/http"
	"strings"
)

// Middleware returns an HTTP middleware that attaches CORS response headers
// when the request Origin matches one of the allowed origins. Preflight
// OPTIONS requests from an allowed origin are short-circuited with 204
// No Content. When the allow list is empty the middleware is a no-op, which
// keeps same-origin deployments unaffected.
func Middleware(allowed []string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, origin := range allowed {
		if origin = strings.TrimSpace(origin); origin != "" {
			set[origin] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := set[origin]; !ok {
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type")
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
