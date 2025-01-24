package xerr

// ErrCode 业务错误码
type ErrCode int

const (
	ParamError             ErrCode = 400 // 参数错误
	UnauthorizedError      ErrCode = 401 // 无权限
	ServerError            ErrCode = 500 // 服务器内部错误
	DbError                ErrCode = 600 // 数据库错误
	CaptchaError           ErrCode = 700 // 验证码错误
	GoogleAuthCodeRequired ErrCode = 701 // 需要google验证码
)

// 通用错误
var (
	ErrParam                  = New(ParamError, "param error")
	ErrUnauthorized           = New(UnauthorizedError, "unauthorized error")
	ErrServer                 = New(ServerError, "server error")
	ErrDB                     = New(DbError, "db error")
	ErrCaptcha                = New(CaptchaError, "captcha error")
	ErrGoogleAuthCodeRequired = New(GoogleAuthCodeRequired, "google auth code required")
)

func (e ErrCode) Int() int {
	return int(e)
}

/*
 * 业务code
 * 1000-1999 会员管理相关
 * 2000-2999 游戏管理相关
 * 3000-3999 运营管理相关
 * 4000-4999 代理管理相关
 * 5000-5999 优惠中心相关
 * 6000-6999 财务管理相关
 * 7000-7999 风控管理相关
 * 8000-8999 报表管理相关
 * 9000-9999 系统管理相关
 * 10000+ 其它
 */

type ErrCodeMessage struct {
	Code ErrCode `json:"code"`
	Msg  string  `json:"msg"`
}
