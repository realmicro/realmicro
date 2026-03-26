package logic

import (
	"context"

	"bootstrap/models"
	"bootstrap/proto"

	mstrings "github.com/realmicro/realmicro/common/util/strings"
	mtime "github.com/realmicro/realmicro/common/util/time"
	log "github.com/sirupsen/logrus"
	"xorm.io/xorm"
)

// Login login to the system, return with login token.
//
// @example {"username": "realmicro", "password": "realmicro111"}
func (s *BootstrapService) Login(ctx context.Context, req *bootstrap.LoginRequest, rsp *bootstrap.LoginResponse) error {
	rsp.Status = &bootstrap.Status{}

	if len(req.Username) == 0 || len(req.Password) == 0 {
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_PARAM_ERROR)
		return nil
	}

	logger := log.WithFields(log.Fields{
		"Module": "Service",
		"Method": "Login",
	})

	account, err := models.GetAccount(func(session *xorm.Session) *xorm.Session {
		return session.Where("username = ?", req.Username)
	})
	if err != nil {
		logger.WithFields(log.Fields{
			"ErrorType": "Database",
			"Function":  "GetAccount",
		}).Error(err)
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_INTERNAL_ERROR)
		return nil
	}
	if account == nil {
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_PARAM_ERROR)
		return nil
	}
	if account.IfBlacklist() {
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_PARAM_ERROR, bootstrap.Msg(bootstrap.AccountStatus(account.Status)))
		return nil
	}
	if !account.CheckPassword(req.Password) {
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_PARAM_ERROR)
		return nil
	}

	account.LastLogin = mtime.Now()
	if err = models.UpdateAccount(func(session *xorm.Session) *xorm.Session {
		return session.ID(account.Id).Cols("last_login")
	}, account); err != nil {
		logger.WithFields(log.Fields{
			"ErrorType": "Database",
			"Function":  "UpdateAccount",
		}).Error(err)
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_INTERNAL_ERROR)
		return nil
	}

	rsp.Token = mstrings.Int2String(account.Id)

	return nil
}
