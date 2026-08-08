package proto

type LoginReq struct {
	Username string `validate:"required"`
	Password string `validate:"required"`
}

type LoginRsp struct {
	Token string `json:"token"`
}

type GetAccountRsp struct {
	Username string `json:"username"`
}

type ChangePasswordReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type IsPasswordUpdatedRsp struct {
	IsUpdated bool `json:"isUpdated"`
}

type AuthConfigRsp struct {
	OIDCEnabled     bool   `json:"oidcEnabled"`
	OIDCReady       bool   `json:"oidcReady"`
	OIDCError       string `json:"oidcError,omitempty"`
	ProviderName    string `json:"providerName"`
	AllowLocalLogin bool   `json:"allowLocalLogin"`
}

type SessionRsp struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username"`
	DisplayName   string `json:"displayName,omitempty"`
	Email         string `json:"email,omitempty"`
	AuthSource    string `json:"authSource"`
	Admin         bool   `json:"admin"`
}
