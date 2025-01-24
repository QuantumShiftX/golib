package googleverifier

import (
	"sync"
)

var (
	recaptchaOnce sync.Once
	recaptcha     *ReCaptchaService

	twoFactorOnce sync.Once
	twoFactor     *TwoFactorAuth
)

// Config 配置结构
type Config struct {
	ReCaptcha struct {
		URL    string
		Secret string
	}
	TwoFactor struct {
		TimeStep   int64
		CodeLength uint32
		WindowSize int
	}
}

// Setup 初始化所有服务
func Setup(cfg *Config, httpClient HTTPClient) {
	SetupReCaptcha(cfg.ReCaptcha.URL, cfg.ReCaptcha.Secret, httpClient)
	SetupTwoFactor(cfg.TwoFactor.TimeStep, cfg.TwoFactor.CodeLength, cfg.TwoFactor.WindowSize)
}

// SetupReCaptcha 初始化 reCAPTCHA 服务
func SetupReCaptcha(url, secret string, httpClient HTTPClient) {
	recaptchaOnce.Do(func() {
		recaptcha = NewReCaptchaService(url, secret, httpClient)
	})
}

// SetupTwoFactor 初始化双因素认证服务
func SetupTwoFactor(timeStep int64, codeLength uint32, windowSize int) {
	twoFactorOnce.Do(func() {
		twoFactor = NewTwoFactorAuth(
			WithTimeStep(timeStep),
			WithCodeLength(codeLength),
			WithWindowSize(windowSize),
		)
	})
}

// GetReCaptcha 获取 reCAPTCHA 服务实例
func GetReCaptcha() *ReCaptchaService {
	return recaptcha
}

// GetTwoFactor 获取双因素认证服务实例
func GetTwoFactor() *TwoFactorAuth {
	return twoFactor
}
