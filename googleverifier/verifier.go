package googleverifier

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// ReCaptcha 相关常量
const (
	ActionLogin    = "login"
	ActionSignup   = "signup"
	ActionResetPwd = "password_reset"
)

// ReCaptchaService 处理 Google reCAPTCHA 验证
type ReCaptchaService struct {
	url    string
	secret string
	client HTTPClient
}

// HTTPClient 接口用于HTTP请求,方便测试时mock
type HTTPClient interface {
	Post(url string, data []byte) ([]byte, error)
}

// NewReCaptchaService 创建新的ReCaptchaService实例
func NewReCaptchaService(url, secret string, client HTTPClient) *ReCaptchaService {
	return &ReCaptchaService{
		url:    url,
		secret: secret,
		client: client,
	}
}

// Verify 验证reCAPTCHA token
func (s *ReCaptchaService) Verify(action, recToken string, minScore float64) (bool, error) {
	req := &ReCaptchaRequest{
		Event: RecapEvent{
			Token:          recToken,
			ExpectedAction: action,
			SiteKey:        s.secret,
		},
	}

	resp, err := s.sendVerifyRequest(req)
	if err != nil {
		return false, fmt.Errorf("recaptcha verification failed: %w", err)
	}

	return s.validateResponse(resp, action, minScore)
}

func (s *ReCaptchaService) sendVerifyRequest(req *ReCaptchaRequest) (*ReCaptchaResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	respData, err := s.client.Post(s.url, data)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	var resp ReCaptchaResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &resp, nil
}

func (s *ReCaptchaService) validateResponse(resp *ReCaptchaResponse, expectedAction string, minScore float64) (bool, error) {
	if resp.Error != nil {
		return false, fmt.Errorf("recaptcha error: %s", resp.Error.Message)
	}

	if !resp.TokenProperties.Valid {
		return false, fmt.Errorf("invalid token: %s", resp.TokenProperties.InvalidReason)
	}

	if resp.TokenProperties.Action != expectedAction {
		return false, fmt.Errorf("action mismatch: expected %s, got %s", expectedAction, resp.TokenProperties.Action)
	}

	return resp.RiskAnalysis.Score >= minScore, nil
}

// TwoFactorAuth 处理双因素认证
type TwoFactorAuth struct {
	// 可配置的参数
	timeStep     int64     // TOTP时间步长(默认60秒)
	codeLength   uint32    // 验证码长度(默认6位)
	windowSize   int       // 时间窗口大小(默认前后1个时间步长)
	randomSource io.Reader // 随机数生成器,便于测试
}

// NewTwoFactorAuth 创建新的TwoFactorAuth实例
func NewTwoFactorAuth(opts ...TwoFactorOption) *TwoFactorAuth {
	auth := &TwoFactorAuth{
		timeStep:     60,
		codeLength:   6,
		windowSize:   1,
		randomSource: rand.Reader,
	}

	for _, opt := range opts {
		opt(auth)
	}

	return auth
}

// TwoFactorOption 定义TwoFactorAuth的可选配置
type TwoFactorOption func(*TwoFactorAuth)

// WithTimeStep 设置TOTP时间步长
func WithTimeStep(seconds int64) TwoFactorOption {
	return func(a *TwoFactorAuth) {
		a.timeStep = seconds
	}
}

// WithCodeLength 设置验证码长度
func WithCodeLength(length uint32) TwoFactorOption {
	return func(a *TwoFactorAuth) {
		a.codeLength = length
	}
}

// WithWindowSize 设置时间窗口大小
func WithWindowSize(size int) TwoFactorOption {
	return func(a *TwoFactorAuth) {
		a.windowSize = size
	}
}

// GenerateSecret 生成随机密钥
func (a *TwoFactorAuth) GenerateSecret() (string, error) {
	const dictionary = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567" // Base32字符集
	bytes := make([]byte, 20)                             // 160位随机数

	if _, err := a.randomSource.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random failed: %w", err)
	}

	for i, b := range bytes {
		bytes[i] = dictionary[b%byte(len(dictionary))]
	}

	return base32.StdEncoding.EncodeToString(bytes), nil
}

// GenerateQRCodeURL 生成用于扫描的二维码URL
func (a *TwoFactorAuth) GenerateQRCodeURL(issuer, accountName, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		issuer,
		accountName,
		secret,
		issuer,
		a.codeLength,
		a.timeStep,
	)
}

// VerifyCode 验证TOTP码
func (a *TwoFactorAuth) VerifyCode(secret string, code int32) bool {
	secretBytes, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	timestamp := time.Now().Unix()
	timeCounter := timestamp / a.timeStep

	// 检查时间窗口内的所有可能值
	for i := -a.windowSize; i <= a.windowSize; i++ {
		if a.generateCode(secretBytes, timeCounter+int64(i)) == code {
			return true
		}
	}

	return false
}

// generateCode 生成特定时间戳的TOTP码
func (a *TwoFactorAuth) generateCode(secret []byte, timeCounter int64) int32 {
	timeBytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		timeBytes[i] = byte(timeCounter & 0xff)
		timeCounter >>= 8
	}

	h := hmac.New(sha1.New, secret)
	h.Write(timeBytes)
	hash := h.Sum(nil)

	offset := hash[len(hash)-1] & 0xf
	truncatedHash := hash[offset : offset+4]
	truncatedHash[0] &= 0x7f // 清除最高位

	code := uint32(truncatedHash[0])<<24 |
		uint32(truncatedHash[1])<<16 |
		uint32(truncatedHash[2])<<8 |
		uint32(truncatedHash[3])

	// 生成指定长度的验证码
	divisor := uint32(1)
	for i := uint32(0); i < a.codeLength; i++ {
		divisor *= 10
	}

	return int32(code % divisor)
}
