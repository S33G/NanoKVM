package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/middleware"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const providerRetryInterval = 30 * time.Second

const oidcBindingCookieName = "nano-kvm-oidc-binding"

const (
	oidcLoginRateWindow    = time.Minute
	maxOIDCLoginsPerWindow = 20
	maxOIDCLoginPeers      = 1024
)

type discoveredProvider struct {
	provider *oidc.Provider
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

type oidcService struct {
	conf       func() config.OIDC
	states     *oidcStateStore
	httpClient *http.Client
	mu         sync.Mutex
	provider   *discoveredProvider
	lastError  time.Time
	rateMu     sync.Mutex
	loginRates map[string]oidcLoginRate
	validate   func(config.OIDC) error
}

type oidcLoginRate struct {
	windowStart time.Time
	count       int
}

type oidcClaims struct {
	Username    string
	DisplayName string
	Email       string
	Groups      []string
	Admin       bool
}

func newOIDCService() *oidcService {
	return &oidcService{
		conf:       func() config.OIDC { return config.GetInstance().OIDC },
		states:     newOIDCStateStore(),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		loginRates: make(map[string]oidcLoginRate),
		validate:   validateConfiguredOIDC,
	}
}

func (s *oidcService) publicStatus() (enabled, ready bool, errorCode string) {
	conf := s.conf()
	if !conf.Enabled {
		return false, false, ""
	}
	if err := s.validate(conf); err != nil {
		return true, false, "invalid_configuration"
	}
	return true, true, ""
}

func (s *oidcService) login(c *gin.Context) {
	conf := s.conf()
	if !conf.Enabled {
		s.redirectError(c, "oidc_disabled")
		return
	}
	if !s.allowLogin(c.RemoteIP(), time.Now()) {
		log.Warn("OIDC login initiation rate limited")
		s.redirectError(c, "oidc_rate_limited")
		return
	}
	if err := s.validate(conf); err != nil {
		log.WithError(err).Warn("OIDC configuration is invalid")
		s.redirectError(c, "oidc_invalid_config")
		return
	}
	provider, err := s.getProvider(c.Request.Context())
	if err != nil {
		log.WithError(err).Warn("OIDC provider discovery failed")
		s.redirectError(c, "oidc_provider_unavailable")
		return
	}
	binding, err := s.browserBinding(c)
	if err != nil {
		log.WithError(err).Warn("OIDC browser binding creation failed")
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	state, nonce, _, challenge, err := s.states.create(binding)
	if err != nil {
		log.WithError(err).Warn("OIDC login transaction creation failed")
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	location := provider.oauth.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	c.Redirect(http.StatusFound, location)
}

func (s *oidcService) callback(c *gin.Context) {
	conf := s.conf()
	if !conf.Enabled {
		s.redirectError(c, "oidc_disabled")
		return
	}
	binding, err := c.Cookie(oidcBindingCookieName)
	if err != nil {
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	txn, ok := s.states.consume(c.Query("state"), binding)
	if !ok {
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	if c.Query("error") != "" {
		s.redirectError(c, "oidc_access_denied")
		return
	}
	code := c.Query("code")
	if code == "" {
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	provider, err := s.getProvider(c.Request.Context())
	if err != nil {
		log.WithError(err).Warn("OIDC provider unavailable during callback")
		s.redirectError(c, "oidc_provider_unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)
	token, err := provider.oauth.Exchange(ctx, code, oauth2.VerifierOption(txn.pkceVerifier))
	if err != nil {
		log.Warn("OIDC authorization code exchange failed")
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	idToken, err := provider.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Warn("OIDC ID token verification failed")
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil || !nonceMatches(claims["nonce"], txn.nonce) ||
		!validAuthorizedParty(claims["azp"], idToken.Audience, conf.ClientID) {
		log.Warn("OIDC nonce or claims validation failed")
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	s.mergeUserInfo(ctx, provider, token, idToken.Subject, claims)
	identity, err := resolveOIDCIdentity(conf, idToken.Subject, claims)
	if err != nil {
		log.WithError(err).Warn("OIDC identity was denied")
		s.redirectError(c, "oidc_access_denied")
		return
	}
	sessionToken, err := middleware.GenerateSessionJWT(middleware.Identity{
		Username: identity.Username, DisplayName: identity.DisplayName, Email: identity.Email,
		AuthSource: "oidc", Issuer: conf.Issuer, Subject: idToken.Subject, Admin: identity.Admin,
	})
	if err != nil {
		log.WithError(err).Warn("OIDC NanoKVM session creation failed")
		s.redirectError(c, "oidc_callback_failed")
		return
	}
	middleware.SetOIDCSessionCookie(c, sessionToken)
	c.Redirect(http.StatusSeeOther, "/#/")
}

func (s *oidcService) getProvider(parent context.Context) (*discoveredProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.provider != nil {
		return s.provider, nil
	}
	if !s.lastError.IsZero() && time.Since(s.lastError) < providerRetryInterval {
		return nil, errors.New("OIDC provider discovery is temporarily unavailable")
	}
	conf := s.conf()
	secret, err := oidcClientSecret(conf)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	ctx = oidc.ClientContext(ctx, s.httpClient)
	provider, err := oidc.NewProvider(ctx, conf.Issuer)
	if err != nil {
		s.lastError = time.Now()
		return nil, err
	}
	discovered := &discoveredProvider{
		provider: provider,
		oauth: oauth2.Config{
			ClientID: conf.ClientID, ClientSecret: secret, Endpoint: provider.Endpoint(), RedirectURL: conf.RedirectURI, Scopes: conf.Scopes,
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: conf.ClientID}),
	}
	s.provider = discovered
	return discovered, nil
}

func (s *oidcService) mergeUserInfo(ctx context.Context, provider *discoveredProvider, token *oauth2.Token, subject string, claims map[string]any) {
	userinfo, err := provider.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil || userinfo.Subject != subject {
		return
	}
	var extra map[string]any
	if err := userinfo.Claims(&extra); err != nil {
		return
	}
	for key, value := range extra {
		if _, exists := claims[key]; !exists {
			claims[key] = value
		}
	}
}

func resolveOIDCIdentity(conf config.OIDC, subject string, claims map[string]any) (oidcClaims, error) {
	if strings.TrimSpace(subject) == "" {
		return oidcClaims{}, errors.New("sub claim is missing")
	}
	username := claimString(claims, conf.UsernameClaim)
	for _, name := range conf.UsernameFallbackClaims {
		if username == "" {
			username = claimString(claims, name)
		}
	}
	if username == "" {
		username = subject
	}
	groups, groupsValid := claimGroups(claims[conf.GroupsClaim])
	if len(conf.AllowedGroups) > 0 && (!groupsValid || !intersects(groups, conf.AllowedGroups)) {
		return oidcClaims{}, errors.New("user is not in an allowed group")
	}
	if conf.RequireEmailVerified {
		verified, ok := claims["email_verified"].(bool)
		if !ok || !verified || claimString(claims, conf.EmailClaim) == "" {
			return oidcClaims{}, errors.New("verified email is required")
		}
	}
	return oidcClaims{
		Username: username, DisplayName: claimString(claims, conf.DisplayNameClaim),
		Email: claimString(claims, conf.EmailClaim), Groups: groups, Admin: intersects(groups, conf.AdminGroups),
	}, nil
}

func claimString(claims map[string]any, name string) string {
	value, ok := claims[name].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func claimGroups(value any) ([]string, bool) {
	switch groups := value.(type) {
	case nil:
		return nil, true
	case string:
		if strings.TrimSpace(groups) == "" {
			return nil, true
		}
		return []string{groups}, true
	case []string:
		return groups, true
	case []any:
		result := make([]string, 0, len(groups))
		for _, group := range groups {
			name, ok := group.(string)
			if !ok || strings.TrimSpace(name) == "" {
				return nil, false
			}
			result = append(result, name)
		}
		return result, true
	default:
		return nil, false
	}
}

func intersects(left, right []string) bool {
	for _, candidate := range left {
		if containsString(right, candidate) {
			return true
		}
	}
	return false
}

func nonceMatches(value any, expected string) bool {
	nonce, ok := value.(string)
	return ok && len(nonce) == len(expected) && subtle.ConstantTimeCompare([]byte(nonce), []byte(expected)) == 1
}

func validAuthorizedParty(value any, audience []string, clientID string) bool {
	azp, present := value.(string)
	if value != nil && (!present || azp == "") {
		return false
	}
	if len(audience) > 1 && !present {
		return false
	}
	return !present || azp == clientID
}

func (s *oidcService) browserBinding(c *gin.Context) (string, error) {
	binding, err := c.Cookie(oidcBindingCookieName)
	if err != nil {
		binding, err = randomURLValue(32)
		if err != nil {
			return "", err
		}
	}
	redirect, _ := url.Parse(s.conf().RedirectURI)
	http.SetCookie(c.Writer, &http.Cookie{
		Name: oidcBindingCookieName, Value: binding, Path: "/api/auth/oidc",
		MaxAge: int(oidcTransactionLifetime.Seconds()), Expires: time.Now().Add(oidcTransactionLifetime),
		HttpOnly: true, Secure: redirect != nil && redirect.Scheme == "https", SameSite: http.SameSiteLaxMode,
	})
	return binding, nil
}

func (s *oidcService) allowLogin(peer string, now time.Time) bool {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	for key, rate := range s.loginRates {
		if now.Sub(rate.windowStart) >= oidcLoginRateWindow {
			delete(s.loginRates, key)
		}
	}
	rate, exists := s.loginRates[peer]
	if !exists {
		if len(s.loginRates) >= maxOIDCLoginPeers {
			return false
		}
		s.loginRates[peer] = oidcLoginRate{windowStart: now, count: 1}
		return true
	}
	if rate.count >= maxOIDCLoginsPerWindow {
		return false
	}
	rate.count++
	s.loginRates[peer] = rate
	return true
}

func (s *oidcService) redirectError(c *gin.Context, code string) {
	c.Redirect(http.StatusSeeOther, "/#/auth/login?"+url.Values{"oidc_error": []string{code}}.Encode())
}
