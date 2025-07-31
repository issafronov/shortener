package trustedsubnet

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	_, trustedNet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("failed to parse CIDR: %v", err)
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mw := TrustedSubnetMiddleware(trustedNet)(nextHandler)

	tests := []struct {
		name       string
		ip         string
		wantStatus int
	}{
		{"Allowed IP in subnet", "192.168.1.100", http.StatusOK},
		{"IP outside subnet", "10.0.0.1", http.StatusForbidden},
		{"Invalid IP", "invalid_ip", http.StatusForbidden},
		{"Empty IP", "", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.ip != "" {
				req.Header.Set("X-Real-IP", tt.ip)
			}

			rec := httptest.NewRecorder()

			mw.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
