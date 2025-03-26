package xerr

// ErrCode 业务错误码
type ErrCode int

const (
	ParamError             ErrCode = 400 // 参数错误
	UnauthorizedError      ErrCode = 401 // 无权限
	ServerError            ErrCode = 500 // network service is congested. please try again later.
	ServerInternalError    ErrCode = 501 // 服务器出错
	DbError                ErrCode = 600 // 数据库错误
	CaptchaError           ErrCode = 700 // 验证码错误
	GoogleAuthCodeRequired ErrCode = 701 // 需要google验证码
)

// 通用错误
var (
	ErrParam                  = New(ParamError, "param error")
	ErrUnauthorized           = New(UnauthorizedError, "unauthorized error")
	ErrorServer               = New(ServerError, "network service is congested. please try again later.")
	ErrorInternalServer       = New(ServerInternalError, "server error")
	ErrDB                     = New(DbError, "db error")
	ErrCaptcha                = New(CaptchaError, "captcha error")
	ErrGoogleAuthCodeRequired = New(GoogleAuthCodeRequired, "google auth code required")
)

func (e ErrCode) Int() int {
	return int(e)
}

func (e ErrCode) Int64() int64 {
	return int64(e)
}

/*
 * 业务code
 */

type ErrCodeMessage struct {
	Code ErrCode `json:"code"`
	Msg  string  `json:"msg"`
}
