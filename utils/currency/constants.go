package currency

import (
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"strings"
)

type Unit int64

const (
	Yuan Unit = 1
	Jiao      = Yuan * 10 // 1元 = 10角
	Fen       = Jiao * 10 // 1角 = 10分
	Li        = Fen * 10  // 1分 = 10厘
	Mao       = Li * 10   // 1厘 = 10毫
	Si        = Mao * 10  // 1毫 = 10丝
	Wei       = Si * 10   // 1丝 = 10微
)

func (c Unit) Decimal() decimal.Decimal {
	return decimal.NewFromInt(int64(c))
}

func (c Unit) Int64() int64 {
	return int64(c)
}

func (c Unit) Int() int {
	return int(c)
}

func (c Unit) Float64() float64 {
	return float64(c)
}

// YuanToWei 元转微
func YuanToWei(c decimal.Decimal) int64 {
	return c.Mul(decimal.NewFromInt(int64(Wei))).IntPart()
}

// WeiToYuan 微转元
func WeiToYuan(c int64) string {
	return decimal.NewFromInt(c).Div(decimal.NewFromInt(int64(Wei))).String()
}

func WeiToYuanFloor(c int64) decimal.Decimal {
	return decimal.NewFromInt(c).Div(decimal.NewFromInt(int64(Wei))).Floor()
}

// YuanToFen 元转分
func YuanToFen(c decimal.Decimal) int64 {
	return c.Mul(decimal.NewFromInt(int64(Fen))).IntPart()
}

// FenToYuan 分转元
func FenToYuan(c int64) string {
	return decimal.NewFromInt(c).Div(decimal.NewFromInt(int64(Fen))).String()
}

// FenToWei 分转微
func FenToWei(c int64) int64 {
	// 1分 = 10000微
	// Wei/Fen = 10000
	ratio := Wei / Fen
	return c * int64(ratio)
}

// WeiToFen 微转分
func WeiToFen(c int64) int64 {
	// 1分 = 10000微
	return decimal.NewFromInt(c).Div(decimal.NewFromInt(int64(Wei / Fen))).IntPart()
}

// ConvertExchangeRateToInt64 将"1000000:790"格式的汇率转换为1 USDT兑换多少指定货币的比率
// 返回适合前端直接使用的int64结果（已乘以Wei）
func ConvertExchangeRateToInt64(exchangeRate string) (int64, error) {
	// 分割字符串
	parts := strings.Split(exchangeRate, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("无效的汇率格式，应为'%d:X'格式", Wei)
	}

	// 解析基础货币值
	baseCurrency, err := cast.ToInt64E(parts[0])
	if err != nil {
		return 0, fmt.Errorf("解析基础货币值失败: %v", err)
	}
	if baseCurrency != int64(Wei) {
		return 0, fmt.Errorf("基础货币值应为%d，收到: %v", Wei, baseCurrency)
	}

	// 解析USDT值
	usdtValue, err := cast.ToInt64E(parts[1])
	if err != nil {
		return 0, fmt.Errorf("解析USDT值失败: %v", err)
	}

	// 避免除以零
	if usdtValue == 0 {
		return 0, fmt.Errorf("USDT值不能为零")
	}

	// 使用预先计算的常量和更简洁的链式操作
	// 计算: (Wei / usdtValue) * Wei
	result := decimal.NewFromInt(baseCurrency).
		Div(decimal.NewFromInt(usdtValue)).
		Mul(decimal.NewFromInt(int64(Wei))).
		IntPart()

	return result, nil
}
