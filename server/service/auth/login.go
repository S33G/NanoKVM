package auth

import (
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Service) Login(c *gin.Context) {
	var req proto.LoginReq
	var rsp proto.Response

	// authentication disabled
	conf := config.GetInstance()
	if conf.Authentication == "disable" {
		middleware.SetSessionCookie(c, "disabled")
		rsp.OkRspWithData(c, &proto.LoginRsp{
			Token: "disabled",
		})
		return
	}
	if conf.OIDC.Enabled && !conf.OIDC.AllowLocalLogin {
		rsp.ErrRsp(c, -6, "local login is disabled")
		return
	}

	clientIP := GetClientIP(c)
	if locked, code, msg := CheckLoginAttempt(clientIP); locked {
		time.Sleep(3 * time.Second)
		rsp.ErrRsp(c, code, msg)
		return
	}

	if err := proto.ParseFormRequest(c, &req); err != nil {
		time.Sleep(3 * time.Second)
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	if ok := CompareAccount(req.Username, req.Password); !ok {
		time.Sleep(2 * time.Second)

		if locked, code, msg := RecordLoginFailure(clientIP); locked {
			rsp.ErrRsp(c, code, msg)
			return
		}

		rsp.ErrRsp(c, -2, "invalid username or password")
		return
	}

	ClearLoginAttempt(clientIP)

	token, err := middleware.GenerateJWT(req.Username)
	if err != nil {
		time.Sleep(1 * time.Second)
		rsp.ErrRsp(c, -3, "generate token failed")
		return
	}
	middleware.SetSessionCookie(c, token)

	rsp.OkRspWithData(c, &proto.LoginRsp{
		Token: token,
	})

	log.Debugf("login success, username: %s", req.Username)
}

func (s *Service) Logout(c *gin.Context) {
	conf := config.GetInstance()

	if conf.JWT.RevokeTokensOnLogout {
		config.RegenerateSecretKey()
	}
	middleware.ClearSessionCookie(c)

	var rsp proto.Response
	rsp.OkRsp(c)
}

func (s *Service) GetPublicConfig(c *gin.Context) {
	conf := config.GetInstance()
	if conf.Authentication == "disable" {
		var rsp proto.Response
		rsp.OkRspWithData(c, &proto.AuthConfigRsp{AllowLocalLogin: true, ProviderName: conf.OIDC.ProviderName})
		return
	}
	enabled, ready, errorCode := s.oidc.publicStatus()
	allowLocalLogin := true
	if conf.OIDC.Enabled {
		allowLocalLogin = conf.OIDC.AllowLocalLogin
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &proto.AuthConfigRsp{
		OIDCEnabled: enabled, OIDCReady: ready, OIDCError: errorCode,
		ProviderName: conf.OIDC.ProviderName, AllowLocalLogin: allowLocalLogin,
	})
}

func (s *Service) GetSession(c *gin.Context) {
	conf := config.GetInstance()
	if conf.Authentication == "disable" {
		var rsp proto.Response
		rsp.OkRspWithData(c, &proto.SessionRsp{Authenticated: true, AuthSource: "disabled"})
		return
	}
	cookie, err := c.Cookie("nano-kvm-token")
	if err != nil {
		var rsp proto.Response
		rsp.OkRspWithData(c, &proto.SessionRsp{Authenticated: false})
		return
	}
	token, err := middleware.ParseJWT(cookie)
	if err != nil || (token.AuthSource == "oidc" && !conf.OIDC.Enabled) {
		var rsp proto.Response
		rsp.OkRspWithData(c, &proto.SessionRsp{Authenticated: false})
		return
	}
	var rsp proto.Response
	rsp.OkRspWithData(c, &proto.SessionRsp{
		Authenticated: true, Username: token.Username, DisplayName: token.DisplayName, Email: token.Email,
		AuthSource: token.AuthSource, Admin: token.Admin,
	})
}

func (s *Service) OIDCLogin(c *gin.Context) {
	if config.GetInstance().Authentication == "disable" {
		s.oidc.redirectError(c, "oidc_disabled")
		return
	}
	s.oidc.login(c)
}

func (s *Service) OIDCCallback(c *gin.Context) {
	if config.GetInstance().Authentication == "disable" {
		s.oidc.redirectError(c, "oidc_disabled")
		return
	}
	s.oidc.callback(c)
}

func (s *Service) GetAccount(c *gin.Context) {
	var rsp proto.Response

	account, err := GetAccount()
	if err != nil {
		rsp.ErrRsp(c, -1, "get account failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetAccountRsp{
		Username: account.Username,
	})
	log.Debugf("get account successful")
}
