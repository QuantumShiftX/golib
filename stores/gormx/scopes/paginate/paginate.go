package paginate

import (
	"gorm.io/gorm"
	"math"
)

// Paginate 分页
func Paginate(pagination *Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if pagination == nil {
			return db
		}

		var (
			tx        = db.Session(&gorm.Session{})
			totalRows int64
			offset    = pagination.Offset()
			limit     = pagination.Limit()
		)

		tx.Model(db.Statement.Model).Count(&totalRows)
		pagination.Total = totalRows
		pagination.TotalPage = int64(math.Ceil(float64(totalRows) / float64(pagination.PageSize)))
		return db.Offset(int(offset)).Limit(int(limit))
	}
}
