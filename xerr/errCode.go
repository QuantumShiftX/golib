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

// 定义预设错误为 *XErr 类型
var (
	ErrParam                  = &XErr{Code: ParamError, Msg: "param error"}
	ErrUnauthorized           = &XErr{Code: UnauthorizedError, Msg: "unauthorized error"}
	ErrorServer               = &XErr{Code: ServerError, Msg: "network service is congested. please try again later."}
	ErrorInternalServer       = &XErr{Code: ServerInternalError, Msg: "server error"}
	ErrDB                     = &XErr{Code: DbError, Msg: "db error"}
	ErrCaptcha                = &XErr{Code: CaptchaError, Msg: "captcha error"}
	ErrGoogleAuthCodeRequired = &XErr{Code: GoogleAuthCodeRequired, Msg: "google auth code required"}
)

func (e ErrCode) Int() int {
	return int(e)
}

func (e ErrCode) Int64() int64 {
	return int64(e)
}
