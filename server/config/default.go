package config

import "github.com/spf13/viper"

var defaultConfig = &Config{
	Proto: "http",
	Host:  "",
	Port: Port{
		Http:  80,
		Https: 443,
	},
	Cert: Cert{
		Crt: "server.crt",
		Key: "server.key",
	},
	Logger: Logger{
		Level: "info",
		File:  "stdout",
	},
	JWT: JWT{
		SecretKey:            "",
		RefreshTokenDuration: 2678400,
		RevokeTokensOnLogout: true,
	},
	OIDC: OIDC{
		ProviderName:           "OIDC",
		Scopes:                 []string{"openid", "profile", "email"},
		UsernameClaim:          "preferred_username",
		UsernameFallbackClaims: []string{"email", "name"},
		DisplayNameClaim:       "name",
		EmailClaim:             "email",
		GroupsClaim:            "groups",
		AllowLocalLogin:        true,
	},
	Stun: "stun.l.google.com:19302",
	Turn: Turn{
		TurnAddr: "",
		TurnUser: "",
		TurnCred: "",
	},
	Authentication: "enable",
	Security: Security{
		LoginLockoutDuration: 0,
		LoginMaxFailures:     5,
	},
}

func checkDefaultValue() {
	if instance.JWT.SecretKey == "" {
		instance.JWT.SecretKey = generateRandomSecretKey()
		instance.JWT.RevokeTokensOnLogout = true
	}

	if instance.JWT.RefreshTokenDuration == 0 {
		instance.JWT.RefreshTokenDuration = 2678400
	}

	if instance.Stun == "" {
		instance.Stun = "stun.l.google.com:19302"
	}

	if instance.Authentication == "" {
		instance.Authentication = "enable"
	}

	if instance.OIDC.ProviderName == "" {
		instance.OIDC.ProviderName = "OIDC"
	}
	if len(instance.OIDC.Scopes) == 0 {
		instance.OIDC.Scopes = []string{"openid", "profile", "email"}
	}
	if instance.OIDC.UsernameClaim == "" {
		instance.OIDC.UsernameClaim = "preferred_username"
	}
	if len(instance.OIDC.UsernameFallbackClaims) == 0 {
		instance.OIDC.UsernameFallbackClaims = []string{"email", "name"}
	}
	if instance.OIDC.DisplayNameClaim == "" {
		instance.OIDC.DisplayNameClaim = "name"
	}
	if instance.OIDC.EmailClaim == "" {
		instance.OIDC.EmailClaim = "email"
	}
	if instance.OIDC.GroupsClaim == "" {
		instance.OIDC.GroupsClaim = "groups"
	}
	if !viper.IsSet("oidc.allowLocalLogin") {
		instance.OIDC.AllowLocalLogin = true
	}

	instance.Hardware = getHardware()
}
