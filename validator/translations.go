package validator

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// 注册自定义错误消息翻译
func registerCustomTranslations() {
	// 获取英文翻译器
	enTrans := translators[LangEN]
	// 获取中文翻译器
	zhTrans := translators[LangZH]

	// 注册英文自定义错误消息
	registerEnglishTranslations(enTrans)
	// 注册中文自定义错误消息
	registerChineseTranslations(zhTrans)
}

// 注册英文自定义错误消息
func registerEnglishTranslations(trans ut.Translator) {
	// 密码验证
	_ = validate.RegisterTranslation("password", trans, func(ut ut.Translator) error {
		return ut.Add("password", "Password must contain at least 8 characters, including uppercase and lowercase letters, numbers, and special characters.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("password", fe.Field())
		return t
	})

	// 字母数字验证
	_ = validate.RegisterTranslation("alpha_num", trans, func(ut ut.Translator) error {
		return ut.Add("alpha_num", "{0} can only contain letters and numbers", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("alpha_num", fe.Field())
		return t
	})

	// 非空验证
	_ = validate.RegisterTranslation("not_empty", trans, func(ut ut.Translator) error {
		return ut.Add("not_empty", "{0} cannot be empty", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("not_empty", fe.Field())
		return t
	})

	// 不能包含特殊字符
	_ = validate.RegisterTranslation("no_special", trans, func(ut ut.Translator) error {
		return ut.Add("no_special", "{0} cannot contain special characters", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("no_special", fe.Field())
		return t
	})

	// 密码格式
	_ = validate.RegisterTranslation("pwd", trans, func(ut ut.Translator) error {
		return ut.Add("pwd", "Password format is incorrect, must contain letters, numbers, and special characters", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("pwd", fe.Field())
		return t
	})

	// IP地址验证
	_ = validate.RegisterTranslation("ip", trans, func(ut ut.Translator) error {
		return ut.Add("ip", "{0} must be a valid IP address", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("ip", fe.Field())
		return t
	})

	// 数字字符串大于
	_ = validate.RegisterTranslation("num_str_gt", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_gt", "{0} must be greater than {1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_gt", fe.Field(), fe.Param())
		return t
	})

	// 数字字符串大于等于
	_ = validate.RegisterTranslation("num_str_gte", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_gte", "{0} must be greater than or equal to {1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_gte", fe.Field(), fe.Param())
		return t
	})

	// 数字字符串小于
	_ = validate.RegisterTranslation("num_str_lt", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_lt", "{0} must be less than {1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_lt", fe.Field(), fe.Param())
		return t
	})

	// 数字字符串小于等于
	_ = validate.RegisterTranslation("num_str_lte", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_lte", "{0} must be less than or equal to {1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_lte", fe.Field(), fe.Param())
		return t
	})

	// 两位小数
	_ = validate.RegisterTranslation("two_decimal_places", trans, func(ut ut.Translator) error {
		return ut.Add("two_decimal_places", "{0} can have at most two decimal places", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("two_decimal_places", fe.Field())
		return t
	})

	// 语言代码
	_ = validate.RegisterTranslation("iso639_1", trans, func(ut ut.Translator) error {
		return ut.Add("iso639_1", "{0} must be a valid ISO 639-1 language code", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("iso639_1", fe.Field())
		return t
	})

	// 时间戳
	_ = validate.RegisterTranslation("valid_timestamp", trans, func(ut ut.Translator) error {
		return ut.Add("valid_timestamp", "{0} must be a valid timestamp", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("valid_timestamp", fe.Field())
		return t
	})
}

// 注册中文自定义错误消息
func registerChineseTranslations(trans ut.Translator) {
	// 密码验证
	_ = validate.RegisterTranslation("password", trans, func(ut ut.Translator) error {
		return ut.Add("password", "密码必须包含大小写字母、数字和特殊字符且至少8位", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("password", fe.Field())
		return t
	})

	// 字母数字验证
	_ = validate.RegisterTranslation("alpha_num", trans, func(ut ut.Translator) error {
		return ut.Add("alpha_num", "{0}只能包含字母和数字", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("alpha_num", fe.Field())
		return t
	})

	// 非空验证
	_ = validate.RegisterTranslation("not_empty", trans, func(ut ut.Translator) error {
		return ut.Add("not_empty", "{0}不能为空", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("not_empty", fe.Field())
		return t
	})

	// 不能包含特殊字符
	_ = validate.RegisterTranslation("no_special", trans, func(ut ut.Translator) error {
		return ut.Add("no_special", "{0}不能包含特殊字符", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("no_special", fe.Field())
		return t
	})

	// 密码格式
	_ = validate.RegisterTranslation("pwd", trans, func(ut ut.Translator) error {
		return ut.Add("pwd", "密码格式不正确，需包含字母、数字和特殊符号", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("pwd", fe.Field())
		return t
	})

	// IP地址验证
	_ = validate.RegisterTranslation("ip", trans, func(ut ut.Translator) error {
		return ut.Add("ip", "{0}必须是有效的IP地址", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("ip", fe.Field())
		return t
	})

	// 数字字符串大于
	_ = validate.RegisterTranslation("num_str_gt", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_gt", "{0}必须大于{1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_gt", fe.Field(), fe.Param())
		return t
	})

	// 数字字符串大于等于
	_ = validate.RegisterTranslation("num_str_gte", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_gte", "{0}必须大于或等于{1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_gte", fe.Field(), fe.Param())
		return t
	})

	// 数字字符串小于
	_ = validate.RegisterTranslation("num_str_lt", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_lt", "{0}必须小于{1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_lt", fe.Field(), fe.Param())
		return t
	})

	// 数字字符串小于等于
	_ = validate.RegisterTranslation("num_str_lte", trans, func(ut ut.Translator) error {
		return ut.Add("num_str_lte", "{0}必须小于或等于{1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("num_str_lte", fe.Field(), fe.Param())
		return t
	})

	// 两位小数
	_ = validate.RegisterTranslation("two_decimal_places", trans, func(ut ut.Translator) error {
		return ut.Add("two_decimal_places", "{0}最多只能有两位小数", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("two_decimal_places", fe.Field())
		return t
	})

	// 语言代码
	_ = validate.RegisterTranslation("iso639_1", trans, func(ut ut.Translator) error {
		return ut.Add("iso639_1", "{0}必须是有效的ISO 639-1语言代码", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("iso639_1", fe.Field())
		return t
	})

	// 时间戳
	_ = validate.RegisterTranslation("valid_timestamp", trans, func(ut ut.Translator) error {
		return ut.Add("valid_timestamp", "{0}必须是有效的时间戳", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("valid_timestamp", fe.Field())
		return t
	})
}
