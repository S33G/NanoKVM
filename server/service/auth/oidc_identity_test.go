package auth

import (
	"reflect"
	"testing"

	"NanoKVM-Server/config"
)

func TestResolveOIDCIdentity(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		claims  map[string]any
		change  func(*config.OIDC)
		want    oidcClaims
		wantErr bool
	}{
		{name: "preferred username and string group", subject: "subject-1", claims: map[string]any{"preferred_username": " alice ", "groups": "admins"}, want: oidcClaims{Username: "alice", Groups: []string{"admins"}, Admin: true}},
		{name: "fallback username", subject: "subject-2", claims: map[string]any{"email": "alice@example.test", "groups": []any{"users"}}, want: oidcClaims{Username: "alice@example.test", Email: "alice@example.test", Groups: []string{"users"}}},
		{name: "subject username fallback", subject: "subject-3", claims: map[string]any{}, want: oidcClaims{Username: "subject-3"}},
		{name: "missing subject", claims: map[string]any{"preferred_username": "alice"}, wantErr: true},
		{name: "missing groups allowed denial", subject: "subject-4", claims: map[string]any{}, change: func(c *config.OIDC) { c.AllowedGroups = []string{"users"} }, wantErr: true},
		{name: "malformed groups allowed denial", subject: "subject-5", claims: map[string]any{"groups": []any{"users", 7}}, change: func(c *config.OIDC) { c.AllowedGroups = []string{"users"} }, wantErr: true},
		{name: "array group allowed", subject: "subject-6", claims: map[string]any{"groups": []any{"users", "operators"}}, change: func(c *config.OIDC) { c.AllowedGroups = []string{"operators"} }, want: oidcClaims{Username: "subject-6", Groups: []string{"users", "operators"}}},
		{name: "wrong allowed group", subject: "subject-7", claims: map[string]any{"groups": []string{"guests"}}, change: func(c *config.OIDC) { c.AllowedGroups = []string{"users"} }, wantErr: true},
		{name: "verified email", subject: "subject-8", claims: map[string]any{"email": "alice@example.test", "email_verified": true}, change: func(c *config.OIDC) { c.RequireEmailVerified = true }, want: oidcClaims{Username: "alice@example.test", Email: "alice@example.test"}},
		{name: "unverified email", subject: "subject-9", claims: map[string]any{"email": "alice@example.test", "email_verified": false}, change: func(c *config.OIDC) { c.RequireEmailVerified = true }, wantErr: true},
		{name: "missing verified flag", subject: "subject-10", claims: map[string]any{"email": "alice@example.test"}, change: func(c *config.OIDC) { c.RequireEmailVerified = true }, wantErr: true},
		{name: "verified but missing email", subject: "subject-11", claims: map[string]any{"email_verified": true}, change: func(c *config.OIDC) { c.RequireEmailVerified = true }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := testOIDCConfig()
			conf.AdminGroups = []string{"admins"}
			if tt.change != nil {
				tt.change(&conf)
			}
			got, err := resolveOIDCIdentity(conf, tt.subject, tt.claims)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveOIDCIdentity() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("resolveOIDCIdentity() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestClaimGroups(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []string
		valid bool
	}{
		{name: "missing", value: nil, valid: true},
		{name: "empty string", value: "  ", valid: true},
		{name: "string", value: "users", want: []string{"users"}, valid: true},
		{name: "string array", value: []string{"users", "admins"}, want: []string{"users", "admins"}, valid: true},
		{name: "JSON array", value: []any{"users", "admins"}, want: []string{"users", "admins"}, valid: true},
		{name: "mixed array", value: []any{"users", 1}, valid: false},
		{name: "blank member", value: []any{"users", " "}, valid: false},
		{name: "object", value: map[string]any{"users": true}, valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := claimGroups(tt.value)
			if valid != tt.valid || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("claimGroups() = (%#v, %v), want (%#v, %v)", got, valid, tt.want, tt.valid)
			}
		})
	}
}

func TestNonceMatches(t *testing.T) {
	if !nonceMatches("expected", "expected") {
		t.Fatal("matching nonce was rejected")
	}
	for _, value := range []any{"wrong", "expecte", nil, 7} {
		if nonceMatches(value, "expected") {
			t.Fatalf("invalid nonce value %#v was accepted", value)
		}
	}
}

func TestValidAuthorizedParty(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		audience []string
		valid    bool
	}{
		{name: "single audience without azp", audience: []string{"nanokvm"}, valid: true},
		{name: "matching azp", value: "nanokvm", audience: []string{"nanokvm"}, valid: true},
		{name: "wrong azp", value: "other-client", audience: []string{"nanokvm"}},
		{name: "malformed azp", value: []string{"nanokvm"}, audience: []string{"nanokvm"}},
		{name: "multiple audiences need azp", audience: []string{"nanokvm", "api"}},
		{name: "multiple audiences matching azp", value: "nanokvm", audience: []string{"nanokvm", "api"}, valid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validAuthorizedParty(tt.value, tt.audience, "nanokvm"); got != tt.valid {
				t.Fatalf("validAuthorizedParty() = %v, want %v", got, tt.valid)
			}
		})
	}
}
