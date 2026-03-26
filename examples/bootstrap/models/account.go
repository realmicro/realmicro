package models

type Account struct {
	Model     `xorm:"extends"`
	Username  string `xorm:"not null default '' VARCHAR(64) unique comment('用户名')" json:"username" mapstructure:"username"`
	Password  string `xorm:"not null default '' VARCHAR(64) comment('密码')" json:"password" mapstructure:"password"`
	Status    int64  `xorm:"not null default 0 INT" json:"status" mapstructure:"status"`
	LastLogin int64  `xorm:"not null default 0 INT" json:"last_login" mapstructure:"last_login"`
}

func CreateAccount(info *Account) (err error) {
	_, err = GetDB().Insert(info)
	return
}

func CreateAccountList(list []*Account) (err error) {
	if len(list) == 0 {
		return
	}
	_, err = GetDB().Insert(list)
	return
}

func DelAccount(filter OrmFilter) (err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	_, err = session.Delete(&Account{})
	return
}

func UpdateAccount(filter OrmFilter, info *Account) (err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	if info == nil {
		info = new(Account)
	}
	_, err = session.Update(info)
	return
}

func UpdateAccountWithAffected(filter OrmFilter, info *Account) (affected int64, err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	if info == nil {
		info = new(Account)
	}
	affected, err = session.Update(info)
	return
}

func GetAccount(filter OrmFilter) (*Account, error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	info := new(Account)
	if has, err := session.Get(info); err != nil {
		return nil, err
	} else {
		if !has {
			return nil, nil
		}
		return info, nil
	}
}

func GetAccountList(filter OrmFilter) (list []*Account, err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	list = make([]*Account, 0)
	err = session.Find(&list)
	return
}

func GetAccountListAndCount(filter OrmFilter) (list []*Account, count int64, err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	list = make([]*Account, 0)
	count, err = session.FindAndCount(&list)
	return
}

func CountAccount(filter OrmFilter) (count int64, err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	return session.Count(&Account{})
}

func SumsAccount(filter OrmFilter, cols []string) (results []float64, err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	return session.Sums(&Account{}, cols...)
}

func SumsIntAccount(filter OrmFilter, cols []string) (results []int64, err error) {
	session := GetDB().NewSession()
	defer session.Close()
	if filter != nil {
		session = filter(session)
	}

	return session.SumsInt(&Account{}, cols...)
}
