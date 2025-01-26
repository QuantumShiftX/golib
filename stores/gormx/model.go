package gormx

import (
	"github.com/QuantumShiftX/golib/metadata"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
	"time"
)

// Model 基础模型
type Model struct {
	CreatedAt int64                 `json:"created_at"`
	UpdatedAt int64                 `json:"updated_at"`
	DeletedAt soft_delete.DeletedAt `json:"deleted_at" gorm:"default:0" `
}

type OperationBaseModel struct {
	OperationTime int64  `gorm:"column:operation_time" json:"operation_time"` // 操作时间，记录修改的时间
	OperatorID    int64  `gorm:"column:operator_id" json:"operator_id"`       // 操作人ID，记录是由谁修改的
	Operator      string `gorm:"column:operator" json:"operator" `            // 操作人，记录是由谁修改的
}

func (base *OperationBaseModel) BeforeSave(tx *gorm.DB) (err error) {
	ctx := tx.Statement.Context
	base.OperationTime = time.Now().Unix()
	if base.OperatorID == 0 {
		base.OperatorID = metadata.GetUidFromCtx(ctx)
	}
	if base.Operator == "" {
		base.Operator = metadata.GetUsernameFromCtx(ctx)
	}
	tx.Statement.SetColumn("operation_time", base.OperationTime)
	tx.Statement.SetColumn("operator_id", base.OperatorID)
	tx.Statement.SetColumn("operator", base.Operator)
	return nil
}
