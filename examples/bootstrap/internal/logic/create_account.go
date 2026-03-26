package logic

import (
	"context"

	"bootstrap/internal/config/mc"
	"bootstrap/models"
	"bootstrap/pkg/utils"
	"bootstrap/proto"

	log "github.com/sirupsen/logrus"
)

// CreateAccount create a new account with username and password.
//
// @example {"account": {"username": "realmicro", "password": "realmicro111"}}
func (s *BootstrapService) CreateAccount(ctx context.Context, req *bootstrap.CreateAccountRequest, rsp *bootstrap.CreateAccountResponse) error {
	rsp.Status = &bootstrap.Status{}

	if req.Account == nil || len(req.Account.Username) == 0 || len(req.Account.Password) == 0 {
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_PARAM_ERROR)
		return nil
	}

	logger := log.WithFields(log.Fields{
		"Module": "Service",
		"Method": "CreateAccount",
	})

	accountBlacklist, err := mc.GetAccountBlacklist()
	if err != nil {
		logger.WithFields(log.Fields{
			"ErrorType": "MicroConfig",
			"Function":  "GetAccountBlacklist",
		}).Error(err)
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_INTERNAL_ERROR)
		return nil
	}
	if utils.LookupString(accountBlacklist.Blacklist, req.Account.Username) {
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_PARAM_ERROR, bootstrap.Msg(bootstrap.AccountStatus_ACCOUNT_STATUS_BLACKLIST))
		return nil
	}

	account := &models.Account{
		Username: req.Account.Username,
		Password: req.Account.Password,
		Status:   int64(bootstrap.AccountStatus_ACCOUNT_STATUS_NORMAL),
	}
	if err := models.CreateAccount(account); err != nil {
		logger.WithFields(log.Fields{
			"ErrorType": "Database",
			"Function":  "CreateAccount",
		}).Error(err)
		bootstrap.ServiceStatus(rsp.Status, bootstrap.StatusCode_STATUS_INTERNAL_ERROR)
		return nil
	}
	rsp.Id = account.Id

	return nil
}
