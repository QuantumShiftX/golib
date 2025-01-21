package genid

import (
	"fmt"
	"github.com/QuantumShiftX/golib/utils/uniqueid"
	"time"
)

// OrderType represents different types of orders
type OrderType string

const (
	// TransferIn represents incoming transfer orders
	TransferIn OrderType = "TI" // 充值
	// TransferOut represents outgoing transfer orders
	TransferOut OrderType = "TO" // 提现
	// DefaultOrder represents default order type
	DefaultOrder OrderType = "DO" // 默认
)

// GenerateOrderNo 生成订单编号。
// 参数:
//
//	orderType: 订单类型。如果为空，默认使用 DefaultOrder。
//
// 返回值:
//
//	string: 生成的订单编号。
//	error: 如果生成唯一ID失败，则返回错误。
func GenerateOrderNo(orderType OrderType) (string, error) {
	//
	if orderType == "" {
		orderType = DefaultOrder
	}

	//
	orderNo, err := uniqueid.GenId()
	if err != nil {
		return "", fmt.Errorf("failed to generate unique ID: %w", err)
	}

	//
	orderNoStr := fmt.Sprintf("%s%s%d",
		orderType,
		time.Now().Format("20060102"),
		orderNo,
	)

	return orderNoStr, nil
}

func GenerateTransferInOrder() (string, error) {
	return GenerateOrderNo(TransferIn)
}

func GenerateTransferOutOrder() (string, error) {
	return GenerateOrderNo(TransferOut)
}

// ValidateOrderNo 验证订单号是否有效。
// 订单号的有效性基于以下两个条件：
// 1. 订单号的长度必须不少于11位。
// 2. 订单号的前缀必须是有效的订单类型前缀之一。
// 参数:
//
//	orderNo - 待验证的订单号字符串。
//
// 返回值:
//
//	如果订单号有效，返回true；否则返回false。
func ValidateOrderNo(orderNo string) bool {
	//
	if len(orderNo) < 11 {
		return false
	}
	//
	prefix := orderNo[:2]
	validPrefixes := map[string]bool{
		string(TransferIn):   true,
		string(TransferOut):  true,
		string(DefaultOrder): true,
	}
	if !validPrefixes[prefix] {
		return false
	}
	//
	return true
}
