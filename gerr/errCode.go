package gerr

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

func (e ErrCode) Int() int {
	return int(e)
}

func (e ErrCode) Int64() int64 {
	return int64(e)
}

// 定义预设错误
var (
	ErrParam                  = New(ParamError, "Invalid parameters. Please try again with correct input.")
	ErrUnauthorized           = New(UnauthorizedError, "Unauthorized. Please log in again.")
	ErrorServer               = New(ServerError, "Network service is congested. Please try again later.")
	ErrorInternalServer       = New(ServerInternalError, "Server error. We're working to resolve this issue.")
	ErrDB                     = New(DbError, "Database error. Please contact support if this persists.")
	ErrCaptcha                = New(CaptchaError, "CAPTCHA verification failed. Please try again.")
	ErrGoogleAuthCodeRequired = New(GoogleAuthCodeRequired, "Google authentication code required.")
)

const (
	SuccessCode int64  = 200
	SuccessMsg  string = "success"
)
