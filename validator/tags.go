package validator

import (
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func registerTags() {
	_ = validate.RegisterValidation("alpha_num", alphaNum)
	_ = validate.RegisterValidation("pwd", pwd)
	_ = validate.RegisterValidation("not_empty", notEmpty)
	_ = validate.RegisterValidation("no_special", noSpecial)
	_ = validate.RegisterValidation("ip", ip)
	_ = validate.RegisterValidation("num_str_gt", numStrGreaterThan)
	_ = validate.RegisterValidation("num_str_gte", numStrGreaterThanOrEqual)
	_ = validate.RegisterValidation("num_str_lt", numStrLessThan)
	_ = validate.RegisterValidation("num_str_lte", numStrLessThanOrEqual)
	_ = validate.RegisterValidation("two_decimal_places", float64WithTwoDecimalPlaces)
	_ = validate.RegisterValidation("password", validatePassword)
	_ = validate.RegisterValidation("iso639_1", validateLanguageCode)
}

// 英文字母加数字
func alphaNum(fl validator.FieldLevel) bool {
	s, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", s)

	return match
}

// 密码，字母/符号/数字 的随机组合
// 字母/符号/数字的随机组合
// 密码必须包含至少一个字母、一个数字和一个特殊符号
func pwd(fl validator.FieldLevel) bool {
	s, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	// 检查长度（可以根据需求调整）
	if len(s) < 8 {
		return false
	}

	// 检查是否包含字母
	hasLetter, err := regexp.MatchString("[a-zA-Z]", s)
	if err != nil || !hasLetter {
		return false
	}

	// 检查是否包含数字
	hasDigit, err := regexp.MatchString("[0-9]", s)
	if err != nil || !hasDigit {
		return false
	}

	// 检查是否包含特殊符号
	hasSymbol, err := regexp.MatchString("[\\W_]", s)
	if err != nil || !hasSymbol {
		return false
	}

	// 检查整体格式 - 只允许字母、数字和特殊符号
	validChars, err := regexp.MatchString("^[a-zA-Z0-9\\W_]+$", s)
	if err != nil {
		return false
	}

	return validChars
}

// 数组不能为空
func notEmpty(fl validator.FieldLevel) bool {
	field := fl.Field()
	//
	switch field.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return field.Len() > 0
	// 其他
	default:
		return false
	}
}

// 不能包含特殊字符
func noSpecial(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	for _, char := range value {
		if char == '!' || char == '@' {
			return false
		}
	}
	return true
}

func ip(fl validator.FieldLevel) bool {
	s, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	// 如果是空字符串，直接返回 true
	if s == "" {
		return true
	}

	// 非空时才验证 IP 格式
	return net.ParseIP(s) != nil
}

func numStrGreaterThan(fl validator.FieldLevel) bool {

	fieldNum, err := decimal.NewFromString(fl.Field().String())
	if err != nil {
		return false
	}

	valueNum, err := decimal.NewFromString(fl.Param())
	if err != nil {
		return false
	}
	return fieldNum.GreaterThan(valueNum)
}

func numStrGreaterThanOrEqual(fl validator.FieldLevel) bool {

	fieldNum, err := decimal.NewFromString(fl.Field().String())
	if err != nil {
		return false
	}

	valueNum, err := decimal.NewFromString(fl.Param())
	if err != nil {
		return false
	}
	return fieldNum.GreaterThanOrEqual(valueNum)
}

func numStrLessThan(fl validator.FieldLevel) bool {

	fieldNum, err := decimal.NewFromString(fl.Field().String())
	if err != nil {
		return false
	}

	valueNum, err := decimal.NewFromString(fl.Param())
	if err != nil {
		return false
	}
	return fieldNum.LessThan(valueNum)
}

func numStrLessThanOrEqual(fl validator.FieldLevel) bool {

	fieldNum, err := decimal.NewFromString(fl.Field().String())
	if err != nil {
		return false
	}

	valueNum, err := decimal.NewFromString(fl.Param())
	if err != nil {
		return false
	}
	return fieldNum.LessThanOrEqual(valueNum)
}

// float64WithTwoDecimalPlaces float64保留2位小数
func float64WithTwoDecimalPlaces(fl validator.FieldLevel) bool {
	value, ok := fl.Field().Interface().(float64)
	if !ok {
		return false
	}
	parts := strings.Split(strconv.FormatFloat(value, 'f', -1, 64), ".")
	if len(parts) == 2 && len(parts[1]) > 2 {
		return false
	}

	return true
}

// validatePassword 验证密码复杂度
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// 至少包含一个大写字母
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	// 至少包含一个小写字母
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	// 至少包含一个数字
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	// 至少包含一个特殊字符
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial
}

// validateLanguageCode 验证语言代码是否符合 ISO 639-1 标准
func validateLanguageCode(fl validator.FieldLevel) bool {
	code := fl.Field().String()
	if code == "" {
		return true
	}

	// ISO 639-1 语言代码为两个小写字母
	match, _ := regexp.MatchString("^[a-z]{2}$", code)
	return match
}

// 支持的语言代码列表(可选，用于进一步验证)
var supportedLanguages = map[string]bool{
	"en": true, // 英语
	"zh": true, // 中文
	"ja": true, // 日语
	"ko": true, // 韩语
	// 可以添加更多支持的语言
}
