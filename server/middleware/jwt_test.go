package middleware

import (
	"testing"
	"time"

	"NanoKVM-Server/config"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseJWTNormalizesLegacyLocalSession(t *testing.T) {
	conf := config.GetInstance()
	conf.JWT.SecretKey = "legacy-session-test-key"
	legacy := struct {
		Username string `json:"username"`
		jwt.RegisteredClaims
	}{
		Username: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, legacy).SignedString([]byte(conf.JWT.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseJWT(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Username != "admin" || claims.AuthSource != "local" {
		t.Fatalf("legacy session parsed as username %q source %q", claims.Username, claims.AuthSource)
	}
}
