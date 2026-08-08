package auth

type Service struct {
	oidc *oidcService
}

func NewService() *Service {
	return &Service{oidc: newOIDCService()}
}
