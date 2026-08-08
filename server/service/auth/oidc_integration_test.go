package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	testClientID     = "nano-kvm-test"
	testClientSecret = "mock-client-secret"
	testKeyID        = "mock-key"
)

type mockOIDCProvider struct {
	server  *httptest.Server
	key     *rsa.PrivateKey
	badKey  *rsa.PrivateKey
	jwksKey *rsa.PrivateKey

	mu            sync.Mutex
	expectedNonce string
	tokenRequests int
	userinfoHits  int
	lastTokenForm url.Values
}

func newMockOIDCProvider(t *testing.T) *mockOIDCProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	badKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	p := &mockOIDCProvider{key: key, badKey: badKey, jwksKey: key}
	p.server = httptest.NewServer(http.HandlerFunc(p.serveHTTP))
	t.Cleanup(p.server.Close)
	return p
}

func (p *mockOIDCProvider) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		writeJSON(w, map[string]any{
			"issuer":                                p.server.URL,
			"authorization_endpoint":                p.server.URL + "/authorize",
			"token_endpoint":                        p.server.URL + "/token",
			"userinfo_endpoint":                     p.server.URL + "/userinfo",
			"jwks_uri":                              p.server.URL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/jwks":
		p.mu.Lock()
		key := p.jwksKey
		p.mu.Unlock()
		writeJSON(w, map[string]any{"keys": []any{rsaJWK(&key.PublicKey)}})
	case "/token":
		p.handleToken(w, r)
	case "/userinfo":
		p.mu.Lock()
		p.userinfoHits++
		p.mu.Unlock()
		writeJSON(w, map[string]any{"sub": "subject-123", "preferred_username": "alice", "email": "alice@example.test"})
	default:
		http.NotFound(w, r)
	}
}

func (p *mockOIDCProvider) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	form := cloneValues(r.Form)
	p.mu.Lock()
	p.tokenRequests++
	p.lastTokenForm = form
	nonce := p.expectedNonce
	p.mu.Unlock()
	code := form.Get("code")
	if code == "exchange-failure" {
		http.Error(w, "exchange rejected", http.StatusBadRequest)
		return
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":            p.server.URL,
		"aud":            testClientID,
		"sub":            "subject-123",
		"iat":            now.Add(-time.Minute).Unix(),
		"exp":            now.Add(5 * time.Minute).Unix(),
		"nonce":          nonce,
		"groups":         []string{"operators", "admins"},
		"email_verified": true,
	}
	key := p.key
	switch code {
	case "invalid-signature":
		key = p.badKey
	case "rotated-key":
		key = p.badKey
	case "wrong-issuer":
		claims["iss"] = "https://wrong-issuer.example.test"
	case "wrong-audience":
		claims["aud"] = "different-client"
	case "wrong-azp":
		claims["azp"] = "different-client"
	case "multiple-audiences-without-azp":
		claims["aud"] = []string{testClientID, "other-api"}
	case "expired":
		claims["exp"] = now.Add(-time.Minute).Unix()
	case "missing-sub":
		delete(claims, "sub")
	case "missing-nonce":
		delete(claims, "nonce")
	case "group-denial":
		claims["groups"] = []string{"guests"}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKeyID
	signed, err := token.SignedString(key)
	if err != nil {
		http.Error(w, "signing failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"access_token": "opaque-access-token",
		"token_type":   "Bearer",
		"expires_in":   300,
		"id_token":     signed,
	})
}

func rsaJWK(key *rsa.PublicKey) map[string]any {
	e := big.NewInt(int64(key.E)).Bytes()
	return map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": testKeyID,
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(e),
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func cloneValues(values url.Values) url.Values {
	clone := make(url.Values, len(values))
	for key, value := range values {
		clone[key] = append([]string(nil), value...)
	}
	return clone
}

func (p *mockOIDCProvider) setNonce(nonce string) {
	p.mu.Lock()
	p.expectedNonce = nonce
	p.mu.Unlock()
}

func (p *mockOIDCProvider) setJWKSKey(key *rsa.PrivateKey) {
	p.mu.Lock()
	p.jwksKey = key
	p.mu.Unlock()
}

func (p *mockOIDCProvider) snapshot() (int, int, url.Values) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tokenRequests, p.userinfoHits, cloneValues(p.lastTokenForm)
}

func integrationOIDCConfig(p *mockOIDCProvider) config.OIDC {
	conf := testOIDCConfig()
	conf.Issuer = p.server.URL
	conf.ClientID = testClientID
	conf.ClientSecret = testClientSecret
	conf.RedirectURI = "http://127.0.0.1:8080/api/auth/oidc/callback"
	conf.AllowedGroups = []string{"operators"}
	conf.AdminGroups = []string{"admins"}
	conf.RequireEmailVerified = true
	return conf
}

func beginOIDCLogin(t *testing.T, service *oidcService, provider *mockOIDCProvider, conf config.OIDC) (state, nonce, challenge string, bindingCookie *http.Cookie) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login?redirect_uri=https://evil.example/callback&next=https://evil.example/", nil)
	service.login(ctx)
	if recorder.Code != http.StatusFound {
		t.Fatalf("login status = %d, want %d", recorder.Code, http.StatusFound)
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorization location: %v", err)
	}
	if location.Scheme+"://"+location.Host+location.Path != provider.server.URL+"/authorize" {
		t.Fatalf("authorization endpoint = %q, want mock provider endpoint", location.Scheme+"://"+location.Host+location.Path)
	}
	query := location.Query()
	if query.Get("redirect_uri") != conf.RedirectURI {
		t.Fatalf("redirect_uri = %q, want configured callback", query.Get("redirect_uri"))
	}
	if query.Get("client_id") != conf.ClientID || query.Get("response_type") != "code" {
		t.Fatal("authorization URL is missing client ID or code response type")
	}
	if !containsString(strings.Fields(query.Get("scope")), "openid") {
		t.Fatal("authorization URL is missing the openid scope")
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q, want S256", query.Get("code_challenge_method"))
	}
	state, nonce, challenge = query.Get("state"), query.Get("nonce"), query.Get("code_challenge")
	if state == "" || nonce == "" || challenge == "" {
		t.Fatal("authorization URL is missing state, nonce, or PKCE challenge")
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == oidcBindingCookieName {
			bindingCookie = cookie
			break
		}
	}
	if bindingCookie == nil || !bindingCookie.HttpOnly || bindingCookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("OIDC login did not set a hardened browser-binding cookie")
	}
	provider.setNonce(nonce)
	return state, nonce, challenge, bindingCookie
}

func performOIDCCallback(service *oidcService, state, code string, bindingCookie *http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	requestURL := "/api/auth/oidc/callback?" + url.Values{
		"state": []string{state},
		"code":  []string{code},
		"next":  []string{"https://evil.example/"},
	}.Encode()
	ctx.Request = httptest.NewRequest(http.MethodGet, requestURL, nil)
	if bindingCookie != nil {
		ctx.Request.AddCookie(bindingCookie)
	}
	service.callback(ctx)
	return recorder
}

func TestOIDCLoginDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := newTestOIDCService()
	s.conf = func() config.OIDC { return config.OIDC{} }
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	s.login(ctx)
	if recorder.Code != http.StatusSeeOther || !strings.Contains(recorder.Header().Get("Location"), "oidc_disabled") {
		t.Fatalf("disabled login response = (%d, %q)", recorder.Code, recorder.Header().Get("Location"))
	}
}

func TestOIDCEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := newMockOIDCProvider(t)
	conf := integrationOIDCConfig(provider)
	nanoConf := config.GetInstance()
	nanoConf.Authentication = "enable"
	nanoConf.JWT.SecretKey = "test-only-nanokvm-signing-key"
	nanoConf.JWT.RefreshTokenDuration = 600
	nanoConf.OIDC = conf

	t.Run("authentication-disabled mode suppresses OIDC", func(t *testing.T) {
		nanoConf.Authentication = "disable"
		defer func() { nanoConf.Authentication = "enable" }()
		service := NewService()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
		service.OIDCLogin(ctx)
		if recorder.Code != http.StatusSeeOther || !strings.Contains(recorder.Header().Get("Location"), "oidc_disabled") {
			t.Fatalf("authentication-disabled OIDC response = (%d, %q)", recorder.Code, recorder.Header().Get("Location"))
		}
	})

	t.Run("local login can be disabled without parsing credentials", func(t *testing.T) {
		localDisabled := conf
		localDisabled.AllowLocalLogin = false
		nanoConf.OIDC = localDisabled
		defer func() { nanoConf.OIDC = conf }()
		service := NewService()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"not-logged"}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		service.Login(ctx)
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || response.Code != -6 {
			t.Fatalf("local-disabled login response = %q", recorder.Body.String())
		}
	})

	t.Run("successful callback and middleware session", func(t *testing.T) {
		service := newTestOIDCService()
		service.conf = func() config.OIDC { return conf }
		state, _, challenge, bindingCookie := beginOIDCLogin(t, service, provider, conf)
		beforeTokens, beforeUserInfo, _ := provider.snapshot()
		crossBrowser := performOIDCCallback(service, state, "success", nil)
		if !strings.Contains(crossBrowser.Header().Get("Location"), "oidc_callback_failed") {
			t.Fatal("callback without the initiating browser binding was accepted")
		}
		response := performOIDCCallback(service, state, "success", bindingCookie)
		if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/#/" {
			t.Fatalf("callback response = (%d, %q), want fixed application redirect", response.Code, response.Header().Get("Location"))
		}
		cookies := response.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("callback set %d cookies, want 1", len(cookies))
		}
		cookie := cookies[0]
		if cookie.Name != "nano-kvm-token" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("session cookie attributes = name %q, HttpOnly %v, SameSite %v", cookie.Name, cookie.HttpOnly, cookie.SameSite)
		}
		afterTokens, afterUserInfo, form := provider.snapshot()
		if afterTokens != beforeTokens+1 || afterUserInfo != beforeUserInfo+1 {
			t.Fatal("successful callback did not call token and UserInfo endpoints once")
		}
		verifier := form.Get("code_verifier")
		digest := sha256.Sum256([]byte(verifier))
		if verifier == "" || base64.RawURLEncoding.EncodeToString(digest[:]) != challenge {
			t.Fatal("token exchange PKCE verifier does not match the authorization challenge")
		}
		if form.Get("redirect_uri") != conf.RedirectURI {
			t.Fatalf("token redirect_uri = %q, want fixed configured callback", form.Get("redirect_uri"))
		}

		router := gin.New()
		router.GET("/protected", middleware.CheckToken(), func(c *gin.Context) {
			identity, ok := c.Get("identity")
			claims, claimsOK := identity.(*middleware.Token)
			if !ok || !claimsOK || claims.Username != "alice" || claims.AuthSource != "oidc" || claims.Subject != "subject-123" || !claims.Admin {
				c.Status(http.StatusInternalServerError)
				return
			}
			c.Status(http.StatusNoContent)
		})
		protected := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.AddCookie(cookie)
		router.ServeHTTP(protected, req)
		if protected.Code != http.StatusNoContent {
			t.Fatalf("protected endpoint status = %d, want %d", protected.Code, http.StatusNoContent)
		}

		wsRouter := gin.New()
		wsRouter.GET("/protected-ws", middleware.CheckToken(), func(c *gin.Context) {
			upgrader := websocket.Upgrader{CheckOrigin: middleware.CheckWebSocketOrigin}
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err == nil {
				_ = conn.Close()
			}
		})
		wsServer := httptest.NewServer(wsRouter)
		defer wsServer.Close()
		headers := http.Header{
			"Cookie": []string{cookie.String()},
			"Origin": []string{wsServer.URL},
		}
		conn, wsResponse, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(wsServer.URL, "http")+"/protected-ws", headers)
		if err != nil {
			status := 0
			if wsResponse != nil {
				status = wsResponse.StatusCode
			}
			t.Fatalf("OIDC session WebSocket handshake failed with status %d: %v", status, err)
		}
		_ = conn.Close()

		rotationState, _, _, rotationBinding := beginOIDCLogin(t, service, provider, conf)
		provider.setJWKSKey(provider.badKey)
		rotated := performOIDCCallback(service, rotationState, "rotated-key", rotationBinding)
		provider.setJWKSKey(provider.key)
		if rotated.Code != http.StatusSeeOther || rotated.Header().Get("Location") != "/#/" {
			t.Fatalf("callback after JWKS rotation = (%d, %q)", rotated.Code, rotated.Header().Get("Location"))
		}

		tokensBeforeReplay, _, _ := provider.snapshot()
		replay := performOIDCCallback(service, state, "success", bindingCookie)
		if replay.Code != http.StatusSeeOther || !strings.Contains(replay.Header().Get("Location"), "oidc_callback_failed") {
			t.Fatalf("state replay response = (%d, %q)", replay.Code, replay.Header().Get("Location"))
		}
		replayedTokens, _, _ := provider.snapshot()
		if replayedTokens != tokensBeforeReplay {
			t.Fatal("replayed state reached the token endpoint")
		}
	})

	tests := []struct {
		name      string
		code      string
		errorCode string
	}{
		{name: "token exchange failure", code: "exchange-failure", errorCode: "oidc_callback_failed"},
		{name: "invalid signature", code: "invalid-signature", errorCode: "oidc_callback_failed"},
		{name: "wrong issuer", code: "wrong-issuer", errorCode: "oidc_callback_failed"},
		{name: "wrong audience", code: "wrong-audience", errorCode: "oidc_callback_failed"},
		{name: "wrong authorized party", code: "wrong-azp", errorCode: "oidc_callback_failed"},
		{name: "multiple audiences without authorized party", code: "multiple-audiences-without-azp", errorCode: "oidc_callback_failed"},
		{name: "expired token", code: "expired", errorCode: "oidc_callback_failed"},
		{name: "missing subject", code: "missing-sub", errorCode: "oidc_access_denied"},
		{name: "missing nonce", code: "missing-nonce", errorCode: "oidc_callback_failed"},
		{name: "group denial", code: "group-denial", errorCode: "oidc_access_denied"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestOIDCService()
			service.conf = func() config.OIDC { return conf }
			state, _, _, bindingCookie := beginOIDCLogin(t, service, provider, conf)
			response := performOIDCCallback(service, state, tt.code, bindingCookie)
			if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), tt.errorCode) {
				t.Fatalf("callback response = (%d, %q), want error %q", response.Code, response.Header().Get("Location"), tt.errorCode)
			}
			if len(response.Result().Cookies()) != 0 {
				t.Fatal("failed callback issued a session cookie")
			}
		})
	}

	t.Run("failure logs redact credentials and codes", func(t *testing.T) {
		var output bytes.Buffer
		previous := log.StandardLogger().Out
		log.SetOutput(&output)
		defer log.SetOutput(previous)

		service := newTestOIDCService()
		service.conf = func() config.OIDC { return conf }
		state, _, _, bindingCookie := beginOIDCLogin(t, service, provider, conf)
		_ = performOIDCCallback(service, state, "exchange-failure", bindingCookie)
		logged := output.String()
		for _, sensitive := range []string{testClientSecret, "exchange-failure", "opaque-access-token"} {
			if strings.Contains(logged, sensitive) {
				t.Fatalf("OIDC failure log contained sensitive authentication data")
			}
		}
	})
}
