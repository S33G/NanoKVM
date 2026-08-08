package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestCheckWebSocketOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "non browser client", want: true},
		{name: "same HTTPS origin", origin: "https://kvm.example.test", want: true},
		{name: "same HTTP origin", origin: "http://kvm.example.test", want: true},
		{name: "same origin with public port", origin: "https://kvm.example.test:8443", host: "kvm.example.test:8443", want: true},
		{name: "different origin", origin: "https://attacker.example.test", want: false},
		{name: "different port", origin: "https://kvm.example.test:8443", want: false},
		{name: "null origin", origin: "null", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://kvm.example.test/api/ws", nil)
			if tt.host != "" {
				req.Host = tt.host
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if got := CheckWebSocketOrigin(req); got != tt.want {
				t.Fatalf("CheckWebSocketOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}
