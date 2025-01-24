package googleverifier

import "time"

// ReCaptcha 请求结构
type ReCaptchaRequest struct {
	Event RecapEvent `json:"event"`
}

type RecapEvent struct {
	Token          string `json:"token"`
	ExpectedAction string `json:"expectedAction"`
	SiteKey        string `json:"siteKey"`
}

// ReCaptcha 响应结构
type ReCaptchaResponse struct {
	Name            string                `json:"name"`
	RiskAnalysis    ReCaptchaRiskAnalysis `json:"riskAnalysis"`
	TokenProperties struct {
		Valid              bool      `json:"valid"`
		InvalidReason      string    `json:"invalidReason"`
		Hostname           string    `json:"hostname"`
		AndroidPackageName string    `json:"androidPackageName"`
		IosBundleId        string    `json:"iosBundleId"`
		Action             string    `json:"action"`
		CreateTime         time.Time `json:"createTime"`
	} `json:"tokenProperties"`
	AccountDefenderAssessment struct {
		Labels []interface{} `json:"labels"`
	} `json:"accountDefenderAssessment"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

type ReCaptchaRiskAnalysis struct {
	Score                  float64       `json:"score"`
	Reasons                []interface{} `json:"reasons"`
	ExtendedVerdictReasons []interface{} `json:"extendedVerdictReasons"`
}
