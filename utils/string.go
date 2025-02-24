package utils

import (
	"encoding/json"
	"strconv"
	"strings"
)

// EnsureJSON 辅助函数：确保字符串是有效的JSON格式
func EnsureJSON(str string) string {
	// 如果是空字符串，返回空JSON对象
	if str == "" {
		return "{}"
	}

	// 尝试解析JSON以验证格式
	var js interface{}
	if err := json.Unmarshal([]byte(str), &js); err != nil {
		// 如果解析失败，返回空JSON对象
		return "{}"
	}

	// 已经是有效的JSON，直接返回
	return str
}

// StrTI64 全数字字符串转int64 (注:非全数字不能使用此函数,因忽略err)
func StrTI64(str string) int64 {
	if str == "" {
		return 0
	}

	// 去除首尾空格
	str = strings.TrimSpace(str)

	result, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return -1
	}

	return result
}
