package trustedsubnet

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет маску подсети
func TrustedSubnetMiddleware(trustedNet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipStr := r.Header.Get("X-Real-IP")

			if trustedNet == nil || ipStr == "" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			ip := net.ParseIP(ipStr)
			if ip == nil || !trustedNet.Contains(ip) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
