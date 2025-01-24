package validator

import (
	"github.com/QuantumShiftX/golib/xerr"
	"github.com/go-playground/validator/v10"
	"sync"
)

var validate *validator.Validate
var once sync.Once

func Init() {
	once.Do(func() {
		if validate == nil {
			validate = validator.New()
		}
		registerTags()
	})
}

// Validate 验证
func Validate(req interface{}) error {
	if err := validate.Struct(req); err != nil {
		return xerr.NewParamErr(err.Error())
	}

	return nil
}
