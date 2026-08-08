package middleware

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/config"
)

type Token struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
	AuthSource  string `json:"authSource,omitempty"`
	Issuer      string `json:"issuer,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Admin       bool   `json:"admin,omitempty"`
	jwt.RegisteredClaims
}

type Identity struct {
	Username    string
	DisplayName string
	Email       string
	AuthSource  string
	Issuer      string
	Subject     string
	Admin       bool
}

func CheckToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowByToken(c) {
			c.Next()
			return
		}

		abortUnauthorized(c)
	}
}

func CheckLoopbackInternalToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowByLoopbackInternalToken(c.Request) {
			c.Next()
			return
		}

		abortUnauthorized(c)
	}
}

func CheckTokenOrLoopbackInternalToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowByToken(c) || allowByLoopbackInternalToken(c.Request) {
			c.Next()
			return
		}

		abortUnauthorized(c)
	}
}

func allowByToken(c *gin.Context) bool {
	conf := config.GetInstance()

	if conf.Authentication == "disable" {
		return true
	}

	cookie, err := c.Cookie("nano-kvm-token")
	if err != nil {
		return false
	}

	claims, err := ParseJWT(cookie)
	if err != nil {
		return false
	}
	if claims.AuthSource == "oidc" && !conf.OIDC.Enabled {
		return false
	}
	c.Set("identity", claims)
	return true
}

func abortUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, "unauthorized")
	c.Abort()
}

func GenerateJWT(username string) (string, error) {
	return GenerateSessionJWT(Identity{Username: username, AuthSource: "local"})
}

func GenerateSessionJWT(identity Identity) (string, error) {
	conf := config.GetInstance()

	expireDuration := time.Duration(conf.JWT.RefreshTokenDuration) * time.Second

	claims := Token{
		Username:    identity.Username,
		DisplayName: identity.DisplayName,
		Email:       identity.Email,
		AuthSource:  identity.AuthSource,
		Issuer:      identity.Issuer,
		Subject:     identity.Subject,
		Admin:       identity.Admin,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return t.SignedString([]byte(config.GetJWTSecretKey()))
}

func ParseJWT(jwtToken string) (*Token, error) {
	t, err := jwt.ParseWithClaims(jwtToken, &Token{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected JWT signing method")
		}
		return []byte(config.GetJWTSecretKey()), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		log.Debugf("parse jwt error: %s", err)
		return nil, err
	}

	if claims, ok := t.Claims.(*Token); ok && t.Valid {
		if claims.AuthSource == "" {
			claims.AuthSource = "local"
		}
		return claims, nil
	}

	return nil, errors.New("invalid JWT claims")
}

func SetSessionCookie(c *gin.Context, token string) {
	setSessionCookie(c, token, false)
}

func SetOIDCSessionCookie(c *gin.Context, token string) {
	redirect, _ := url.Parse(config.GetInstance().OIDC.RedirectURI)
	setSessionCookie(c, token, redirect != nil && redirect.Scheme == "https")
}

func setSessionCookie(c *gin.Context, token string, forceSecure bool) {
	conf := config.GetInstance()
	maxAge := int(conf.JWT.RefreshTokenDuration)
	secure := forceSecure || conf.Proto == "https" || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "nano-kvm-token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "nano-kvm-token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   config.GetInstance().Proto == "https" || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https"),
		SameSite: http.SameSiteLaxMode,
	})
}
