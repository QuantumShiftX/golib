package cryptox

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

// PasswordConfig 密码生成配置
type PasswordConfig struct {
	Length       int  // 密码长度
	Numbers      bool // 是否包含数字
	LowerLetters bool // 是否包含小写字母
	UpperLetters bool // 是否包含大写字母
	SpecialChars bool // 是否包含特殊字符
	MinNumbers   int  // 最少数字数量
	MinLowerCase int  // 最少小写字母数量
	MinUpperCase int  // 最少大写字母数量
	MinSpecial   int  // 最少特殊字符数量
}

const (
	numbers      = "0123456789"
	lowerLetters = "abcdefghijklmnopqrstuvwxyz"
	upperLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// GeneratePassword 生成密码
func GeneratePassword(config PasswordConfig) (string, error) {
	if !isValidConfig(config) {
		return "", fmt.Errorf("invalid password configuration")
	}

	// 构建字符集
	var charSet string
	if config.Numbers {
		charSet += numbers
	}
	if config.LowerLetters {
		charSet += lowerLetters
	}
	if config.UpperLetters {
		charSet += upperLetters
	}
	if config.SpecialChars {
		charSet += specialChars
	}

	// 生成密码
	password := make([]byte, config.Length)
	for i := 0; i < config.Length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
		if err != nil {
			return "", err
		}
		password[i] = charSet[n.Int64()]
	}

	// 确保满足最小要求
	if !meetMinimumRequirements(string(password), config) {
		return GeneratePassword(config)
	}

	return string(password), nil
}

// GenerateSimplePassword 生成简单密码 (仅包含字母和数字)
func GenerateSimplePassword(length int) (string, error) {
	config := PasswordConfig{
		Length:       length,
		Numbers:      true,
		LowerLetters: true,
		UpperLetters: true,
		MinNumbers:   1,
		MinLowerCase: 1,
		MinUpperCase: 1,
	}
	return GeneratePassword(config)
}

// GenerateStrongPassword 生成强密码 (包含所有字符类型)
func GenerateStrongPassword(length int) (string, error) {
	config := PasswordConfig{
		Length:       length,
		Numbers:      true,
		LowerLetters: true,
		UpperLetters: true,
		SpecialChars: true,
		MinNumbers:   2,
		MinLowerCase: 2,
		MinUpperCase: 2,
		MinSpecial:   2,
	}
	return GeneratePassword(config)
}

// isValidConfig 验证配置是否有效
func isValidConfig(config PasswordConfig) bool {
	if config.Length < 1 {
		return false
	}

	minRequired := config.MinNumbers + config.MinLowerCase +
		config.MinUpperCase + config.MinSpecial

	if minRequired > config.Length {
		return false
	}

	return true
}

// meetMinimumRequirements 检查是否满足最小要求
func meetMinimumRequirements(password string, config PasswordConfig) bool {
	var nums, lower, upper, special int

	for _, char := range password {
		switch {
		case unicode.IsNumber(char):
			nums++
		case unicode.IsLower(char):
			lower++
		case unicode.IsUpper(char):
			upper++
		case strings.ContainsRune(specialChars, char):
			special++
		}
	}

	return nums >= config.MinNumbers &&
		lower >= config.MinLowerCase &&
		upper >= config.MinUpperCase &&
		special >= config.MinSpecial
}
