package bootstrap

var StatusCodeMsgMap = map[StatusCode]string{
	StatusCode_STATUS_INTERNAL_ERROR: "内部错误",
	StatusCode_STATUS_PARAM_ERROR:    "参数错误",
}

func (x StatusCode) ErrMsg() string {
	msg, ok := StatusCodeMsgMap[x]
	if ok {
		return msg
	}
	return ""
}

func ServiceStatus(status *Status, statusCode StatusCode, msg ...string) {
	status.Code = statusCode
	status.Msg = statusCode.ErrMsg()
	if len(msg) > 0 {
		status.Msg = msg[0]
	}
}

var AccountStatusMsgMap = map[AccountStatus]string{
	AccountStatus_ACCOUNT_STATUS_NORMAL:    "用户正常",
	AccountStatus_ACCOUNT_STATUS_BLACKLIST: "用户已拉黑",
}

func (x AccountStatus) Msg() (msg string) {
	msg, ok := AccountStatusMsgMap[x]
	if ok {
		return msg
	}
	return
}

func Msg(v interface{}) string {
	switch v.(type) {
	case AccountStatus:
		return v.(AccountStatus).Msg()
	}
	return ""
}
