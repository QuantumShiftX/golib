package scopes

import (
	"fmt"
	"github.com/QuantumShiftX/golib/utils"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/stringx"
	"gorm.io/gorm"
	"reflect"
	"strings"
)

// Equal 等于
func Equal(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" = ?", value)
	}
}

// Equal2 根据条件判断是否启用等于查询
func Equal2(field string, value any, apply bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if apply {
			return db.Where(field+" = ?", value)
		}

		return db
	}
}

// NotEqual 不等于
func NotEqual(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if isZero(value) {
			return db
		}
		return db.Where(field+" != ?", value)
	}
}

// Like 模糊查询
func Like(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if s, ok := value.(string); ok && stringx.HasEmpty(s) {
			return db
		}

		return db.Where(field+" LIKE ?", fmt.Sprintf("%%%v%%", value))
	}
}

// ILike 模糊查询(不区分大小写)
func ILike(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if s, ok := value.(string); ok && stringx.HasEmpty(s) {
			return db
		}

		return db.Where(field+" ILIKE ?", fmt.Sprintf("%%%v%%", value))
	}
}

// In in查询
func In[T comparable](field string, value []T) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if value == nil || len(value) == 0 {
			return db
		}

		if len(value) == 1 {
			return db.Where(field+" = ?", value[0])
		}

		return db.Where(field+" IN ?", value)
	}
}

// NotIn not in查询
func NotIn[T comparable](field string, value []T) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if value == nil || len(value) == 0 {
			return db
		}

		if len(value) == 1 {
			return db.Where(field+"  != ?", value[0])
		}

		return db.Where(field+" NOT IN ?", value)
	}
}

// GT 大于
func GT(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" > ?", value)
	}
}

// GTE 大于等于
func GTE(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" >= ?", value)
	}
}

// LT 小于
func LT(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" < ?", value)
	}
}

// LTE 小于等于
func LTE(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" <= ?", value)
	}
}

// Between between查询
func Between[T comparable](field string, start, end T) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" BETWEEN ? AND ?", start, end)
	}
}

// Between2 范围查询 scope:[true:true]=>[start,end] scope:[false:true]=>[-∞,end] scope:[true:false]=>[start,+∞]
func Between2[T any](field string, start, end T, startScope, endScope bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch {
		case startScope && endScope:
			return db.Where(field+" BETWEEN ? AND ?", start, end)
		case !startScope && endScope:
			return db.Where(field+" <= ?", end)
		case startScope && !endScope:
			return db.Where(field+" >= ?", start)
		}

		return db
	}
}

// NotBetween not between查询
func NotBetween[T comparable](field string, start, end T) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" NOT BETWEEN ? AND ?", start, end)
	}
}

// Select 筛选字段
func Select(fields ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Select(fields)
	}
}

// JsonArrayContains 包含
func JsonArrayContains(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" @> ?", value)
	}
}

// JsonArrayOr json数组or查询
func JsonArrayOr[T any](field string, values ...T) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(values) == 0 {
			return db
		}

		var (
			conditions []string
			args       []interface{}
		)

		for _, value := range values {
			jsonValue, err := jsonx.Marshal(value)
			if err != nil {
				return db
			}
			conditions = append(conditions, fmt.Sprintf("%s @> ?", field))
			args = append(args, jsonValue)
		}

		orCondition := strings.Join(conditions, " OR ")
		return db.Where(orCondition, args...)
	}
}

func OrderBy(field, order string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if field == "" {
			return db
		}
		order = utils.Ternary(order != "", order, "asc")
		return db.Order(field + " " + order)
	}
}

// isZero 检查值是否为零值或空值
func isZero(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
	}
	return false
}
