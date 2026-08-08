package auth

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"NanoKVM-Server/config"
)

func validateOIDCConfig(conf config.OIDC) error {
	if !conf.Enabled {
		return nil
	}
	if conf.ClientID == "" {
		return errors.New("clientId is required")
	}
	if conf.ClientSecret != "" && conf.ClientSecretFile != "" {
		return errors.New("clientSecret and clientSecretFile are mutually exclusive")
	}
	if conf.ClientSecret == "" && conf.ClientSecretFile == "" {
		return errors.New("a client secret is required")
	}
	if _, err := validateOIDCURL("issuer", conf.Issuer); err != nil {
		return err
	}
	if _, err := validateOIDCURL("redirectUri", conf.RedirectURI); err != nil {
		return err
	}
	if !containsString(conf.Scopes, "openid") {
		return errors.New("scopes must include openid")
	}
	if conf.UsernameClaim == "" || conf.GroupsClaim == "" || conf.EmailClaim == "" {
		return errors.New("OIDC claim names must not be empty")
	}
	_, err := oidcClientSecret(conf)
	return err
}

func validateConfiguredOIDC(conf config.OIDC) error {
	return validateConfiguredOIDCFile(conf, config.ConfigurationFile)
}

func validateConfiguredOIDCFile(conf config.OIDC, path string) error {
	if err := validateOIDCConfig(conf); err != nil || !conf.Enabled || conf.ClientSecret == "" {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return errors.New("server.yaml permissions could not be verified")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("server.yaml must be a protected regular file when clientSecret is used")
	}
	return nil
}

func validateOIDCURL(name, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, fmt.Errorf("%s must be an absolute URL", name)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHost(u.Hostname())) {
		return nil, fmt.Errorf("%s must use HTTPS except on loopback", name)
	}
	if name == "issuer" && u.RawQuery != "" {
		return nil, errors.New("issuer must not contain a query")
	}
	return u, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func oidcClientSecret(conf config.OIDC) (string, error) {
	if conf.ClientSecretFile == "" {
		return conf.ClientSecret, nil
	}
	info, err := os.Stat(conf.ClientSecretFile)
	if err != nil {
		return "", fmt.Errorf("client secret file is unavailable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("client secret file must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("client secret file must not be accessible by group or other users")
	}
	secret, err := os.ReadFile(conf.ClientSecretFile)
	if err != nil {
		return "", fmt.Errorf("client secret file is unavailable: %w", err)
	}
	value := strings.TrimSpace(string(secret))
	if value == "" {
		return "", errors.New("client secret file is empty")
	}
	return value, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
