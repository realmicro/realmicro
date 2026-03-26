package mc

import "strings"

const (
	AccountBlacklistConfigPath = "account/blacklist"
)

type AccountBlacklist struct {
	Blacklist []string `json:"blacklist"`
}

func GetAccountBlacklist(args ...string) (*AccountBlacklist, error) {
	info := new(AccountBlacklist)
	ps := strings.Split(AccountBlacklistConfigPath, "/")
	if len(args) > 0 {
		ps = append(ps, args...)
	}
	if err := DefaultConfig.cfg.Get(ps...).Scan(info); err != nil {
		return nil, err
	}
	return info, nil
}
