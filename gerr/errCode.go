package gerr

import "google.golang.org/grpc/codes"

// ErrCode 业务错误码
type ErrCode codes.Code

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
	ErrParam                  = &GError{Code: codes.Code(ParamError), Msg: "Invalid parameters. Please try again with correct input."}
	ErrUnauthorized           = &GError{Code: codes.Code(UnauthorizedError), Msg: "Unauthorized. Please log in again."}
	ErrorServer               = &GError{Code: codes.Code(ServerError), Msg: "Network service is congested. Please try again later."}
	ErrorInternalServer       = &GError{Code: codes.Code(ServerInternalError), Msg: "Server error. We're working to resolve this issue."}
	ErrDB                     = &GError{Code: codes.Code(DbError), Msg: "Database error. Please contact support if this persists."}
	ErrCaptcha                = &GError{Code: codes.Code(CaptchaError), Msg: "CAPTCHA verification failed. Please try again."}
	ErrGoogleAuthCodeRequired = &GError{Code: codes.Code(GoogleAuthCodeRequired), Msg: "Google authentication code required."}
)

func (e ErrCode) Int() int {
	return int(e)
}

func (e ErrCode) Int64() int64 {
	return int64(e)
}

const (
	SuccessCode int64  = 200
	SuccessMsg  string = "success"
)
