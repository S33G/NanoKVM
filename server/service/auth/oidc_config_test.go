package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/config"
)

func testOIDCConfig() config.OIDC {
	return config.OIDC{
		Enabled:                true,
		Issuer:                 "https://idp.example.test",
		ClientID:               "nano-kvm",
		ClientSecret:           "client-secret",
		RedirectURI:            "https://kvm.example.test/api/auth/oidc/callback",
		Scopes:                 []string{"openid", "profile", "email"},
		UsernameClaim:          "preferred_username",
		UsernameFallbackClaims: []string{"email", "name"},
		EmailClaim:             "email",
		GroupsClaim:            "groups",
	}
}

func newTestOIDCService() *oidcService {
	s := newOIDCService()
	s.validate = validateOIDCConfig
	return s
}

func TestValidateOIDCConfigDisabled(t *testing.T) {
	if err := validateOIDCConfig(config.OIDC{}); err != nil {
		t.Fatalf("disabled configuration returned an error: %v", err)
	}
	s := newTestOIDCService()
	s.conf = func() config.OIDC { return config.OIDC{} }
	enabled, ready, code := s.publicStatus()
	if enabled || ready || code != "" {
		t.Fatalf("publicStatus() = (%v, %v, %q), want (false, false, empty)", enabled, ready, code)
	}
	invalid := testOIDCConfig()
	invalid.ClientID = ""
	s.conf = func() config.OIDC { return invalid }
	enabled, ready, code = s.publicStatus()
	if !enabled || ready || code != "invalid_configuration" {
		t.Fatalf("invalid publicStatus() = (%v, %v, %q)", enabled, ready, code)
	}
}

func TestValidateOIDCConfig(t *testing.T) {
	tests := []struct {
		name   string
		change func(*config.OIDC)
		want   string
	}{
		{name: "valid"},
		{name: "missing client ID", change: func(c *config.OIDC) { c.ClientID = "" }, want: "clientId"},
		{name: "missing secret", change: func(c *config.OIDC) { c.ClientSecret = "" }, want: "client secret"},
		{name: "two secrets", change: func(c *config.OIDC) { c.ClientSecretFile = "also-a-file" }, want: "mutually exclusive"},
		{name: "insecure issuer", change: func(c *config.OIDC) { c.Issuer = "http://idp.example.test" }, want: "issuer must use HTTPS"},
		{name: "issuer query", change: func(c *config.OIDC) { c.Issuer = "https://idp.example.test?tenant=bad" }, want: "issuer must not contain a query"},
		{name: "insecure redirect", change: func(c *config.OIDC) { c.RedirectURI = "http://kvm.example.test/callback" }, want: "redirectUri must use HTTPS"},
		{name: "missing openid scope", change: func(c *config.OIDC) { c.Scopes = []string{"profile"} }, want: "openid"},
		{name: "missing username claim", change: func(c *config.OIDC) { c.UsernameClaim = "" }, want: "claim names"},
		{name: "missing groups claim", change: func(c *config.OIDC) { c.GroupsClaim = "" }, want: "claim names"},
		{name: "missing email claim", change: func(c *config.OIDC) { c.EmailClaim = "" }, want: "claim names"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := testOIDCConfig()
			if tt.change != nil {
				tt.change(&conf)
			}
			err := validateOIDCConfig(conf)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("validateOIDCConfig() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateOIDCConfig() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

func TestValidateOIDCURLSecurity(t *testing.T) {
	tests := []struct {
		raw   string
		valid bool
	}{
		{raw: "https://idp.example.test/tenant", valid: true},
		{raw: "http://localhost:8080/issuer", valid: true},
		{raw: "http://127.0.0.1:8080/issuer", valid: true},
		{raw: "http://[::1]:8080/issuer", valid: true},
		{raw: "http://idp.example.test", valid: false},
		{raw: "ftp://localhost/issuer", valid: false},
		{raw: "/relative", valid: false},
		{raw: "https://user@idp.example.test", valid: false},
		{raw: "https://idp.example.test/#fragment", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			_, err := validateOIDCURL("issuer", tt.raw)
			if (err == nil) != tt.valid {
				t.Fatalf("validateOIDCURL() error = %v, valid = %v", err, tt.valid)
			}
		})
	}
}

func TestOIDCClientSecretFile(t *testing.T) {
	dir := t.TempDir()
	secure := filepath.Join(dir, "secret")
	if err := os.WriteFile(secure, []byte("  file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := testOIDCConfig()
	conf.ClientSecret = ""
	conf.ClientSecretFile = secure
	got, err := oidcClientSecret(conf)
	if err != nil || got != "file-secret" {
		t.Fatalf("oidcClientSecret() did not return the trimmed file value: %v", err)
	}
	if err := validateOIDCConfig(conf); err != nil {
		t.Fatalf("validateOIDCConfig() with secure secret file: %v", err)
	}

	tests := []struct {
		name string
		path func(*testing.T) string
		want string
	}{
		{name: "missing", path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") }, want: "unavailable"},
		{name: "directory", path: func(t *testing.T) string { return t.TempDir() }, want: "regular file"},
		{name: "empty", path: func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "empty")
			if err := os.WriteFile(p, []byte(" \n"), 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		}, want: "empty"},
		{name: "overpermissive", path: func(t *testing.T) string {
			p := filepath.Join(t.TempDir(), "secret")
			if err := os.WriteFile(p, []byte("value"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(p, 0o644); err != nil {
				t.Fatal(err)
			}
			return p
		}, want: "group or other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := testOIDCConfig()
			conf.ClientSecret = ""
			conf.ClientSecretFile = tt.path(t)
			_, err := oidcClientSecret(conf)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("oidcClientSecret() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

func TestInlineSecretRequiresProtectedConfigurationFile(t *testing.T) {
	conf := testOIDCConfig()
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if err := validateConfiguredOIDCFile(conf, missing); err == nil {
		t.Fatal("inline secret accepted when configuration permissions could not be verified")
	}

	path := filepath.Join(t.TempDir(), "server.yaml")
	if err := os.WriteFile(path, []byte("oidc: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateConfiguredOIDCFile(conf, path); err == nil {
		t.Fatal("inline secret accepted from an overpermissive configuration file")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateConfiguredOIDCFile(conf, path); err != nil {
		t.Fatalf("inline secret rejected from a protected configuration file: %v", err)
	}

	symlink := filepath.Join(t.TempDir(), "server.yaml")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatal(err)
	}
	if err := validateConfiguredOIDCFile(conf, symlink); err == nil {
		t.Fatal("inline secret accepted from a symlinked configuration file")
	}
}
