package validator

import (
	"errors"
	"github.com/QuantumShiftX/golib/xerr"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTrans "github.com/go-playground/validator/v10/translations/en"
	zhTrans "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"strings"
	"sync"
)

var validate *validator.Validate
var once sync.Once
var universalTranslator *ut.UniversalTranslator
var translators map[string]ut.Translator

// 支持的语言
const (
	LangEN = "en" // 英语（默认）
	LangZH = "zh" // 中文
)

func Init() {
	once.Do(func() {
		if validate == nil {
			validate = validator.New()

			// 支持从多种tag中获取字段名
			validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
				// 尝试从各种标签中获取字段名
				var name string
				tags := []string{"json", "form", "path", "header", "uri", "query"}

				for _, tag := range tags {
					name = fld.Tag.Get(tag)
					if name != "" {
						// 处理有选项的标签, 如 `json:"name,omitempty"`
						if comma := strings.Index(name, ","); comma != -1 {
							name = name[:comma]
						}
						break
					}
				}

				// 如果没有找到标签，则使用字段名
				if name == "" {
					name = fld.Name
				}

				return name
			})
		}

		// 设置翻译器
		english := en.New()
		chinese := zh.New()

		// 第一个参数是回退的语言环境, 这里设为英语
		universalTranslator = ut.New(english, english, chinese)

		// 初始化翻译器映射
		translators = make(map[string]ut.Translator)

		// 获取英语翻译器
		enTranslator, _ := universalTranslator.GetTranslator(LangEN)
		translators[LangEN] = enTranslator

		// 获取中文翻译器
		zhTranslator, _ := universalTranslator.GetTranslator(LangZH)
		translators[LangZH] = zhTranslator

		// 注册英语翻译器的默认翻译
		_ = enTrans.RegisterDefaultTranslations(validate, enTranslator)

		// 注册中文翻译器的默认翻译
		_ = zhTrans.RegisterDefaultTranslations(validate, zhTranslator)

		// 注册自定义错误消息
		registerCustomTranslations()

		// 注册自定义验证标签
		registerTags()
	})
}

// Validate 使用默认语言(英语)验证
func Validate(req interface{}) error {
	return ValidateWithLang(req, LangEN)
}

// ValidateZH 使用中文验证
func ValidateZH(req interface{}) error {
	return ValidateWithLang(req, LangZH)
}

// ValidateWithLang 使用指定语言验证
func ValidateWithLang(req interface{}, lang string) error {
	// 检查语言是否支持，不支持则使用默认语言(英语)
	translator, ok := translators[lang]
	if !ok {
		translator = translators[LangEN]
	}

	if err := validate.Struct(req); err != nil {
		// 将验证错误转换为翻译后的错误信息
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			// 获取第一个错误的翻译
			if len(errs) > 0 {
				return xerr.NewParamErr(errs[0].Translate(translator))
			}
		}
		// 如果不是标准验证错误，则返回原始错误
		return xerr.NewParamErr(err.Error())
	}
	return nil
}

// ValidateAllErrors 返回所有错误
func ValidateAllErrors(req interface{}, lang string) []string {
	translator, ok := translators[lang]
	if !ok {
		translator = translators[LangEN]
	}

	if err := validate.Struct(req); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			var errMsgs []string
			for _, e := range errs {
				errMsgs = append(errMsgs, e.Translate(translator))
			}
			return errMsgs
		}
		return []string{err.Error()}
	}
	return nil
}
