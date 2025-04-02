package currency

import "github.com/shopspring/decimal"

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
