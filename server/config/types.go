package config

type Config struct {
	Proto          string   `yaml:"proto"`
	Host           string   `yaml:"host"`
	Port           Port     `yaml:"port"`
	Cert           Cert     `yaml:"cert"`
	Logger         Logger   `yaml:"logger"`
	Authentication string   `yaml:"authentication"`
	JWT            JWT      `yaml:"jwt"`
	OIDC           OIDC     `yaml:"oidc"`
	Stun           string   `yaml:"stun"`
	Turn           Turn     `yaml:"turn"`
	Security       Security `yaml:"security"`

	Hardware Hardware `yaml:"-"`
}

type Logger struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type Port struct {
	Http  int `yaml:"http"`
	Https int `yaml:"https"`
}

type Cert struct {
	Crt string `yaml:"crt"`
	Key string `yaml:"key"`
}

type JWT struct {
	SecretKey            string `yaml:"secretKey"`
	RefreshTokenDuration uint64 `yaml:"refreshTokenDuration"`
	RevokeTokensOnLogout bool   `yaml:"revokeTokensOnLogout"`
}

type OIDC struct {
	Enabled                bool     `yaml:"enabled"`
	ProviderName           string   `yaml:"providerName"`
	Issuer                 string   `yaml:"issuer"`
	ClientID               string   `yaml:"clientId"`
	ClientSecret           string   `yaml:"clientSecret"`
	ClientSecretFile       string   `yaml:"clientSecretFile"`
	RedirectURI            string   `yaml:"redirectUri"`
	Scopes                 []string `yaml:"scopes"`
	UsernameClaim          string   `yaml:"usernameClaim"`
	UsernameFallbackClaims []string `yaml:"usernameFallbackClaims"`
	DisplayNameClaim       string   `yaml:"displayNameClaim"`
	EmailClaim             string   `yaml:"emailClaim"`
	GroupsClaim            string   `yaml:"groupsClaim"`
	AdminGroups            []string `yaml:"adminGroups"`
	AllowedGroups          []string `yaml:"allowedGroups"`
	AllowLocalLogin        bool     `yaml:"allowLocalLogin"`
	RequireEmailVerified   bool     `yaml:"requireEmailVerified"`
}

type Turn struct {
	TurnAddr string `yaml:"turnAddr"`
	TurnUser string `yaml:"turnUser"`
	TurnCred string `yaml:"turnCred"`
}

type Security struct {
	LoginLockoutDuration int `yaml:"loginLockoutDuration"`
	LoginMaxFailures     int `yaml:"loginMaxFailures"`
}

type Hardware struct {
	Version      HWVersion `yaml:"-"`
	GPIOReset    string    `yaml:"-"`
	GPIOPower    string    `yaml:"-"`
	GPIOPowerLED string    `yaml:"-"`
	GPIOHDDLed   string    `yaml:"-"`
}
