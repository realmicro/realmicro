package models

import "bootstrap/proto"

func (a *Account) IfBlacklist() bool {
	return a.Status == int64(bootstrap.AccountStatus_ACCOUNT_STATUS_BLACKLIST)
}

func (a *Account) CheckPassword(password string) bool {
	return a.Password == password
}
