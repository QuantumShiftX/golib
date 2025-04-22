package timec

import (
	"github.com/QuantumShiftX/farms-pkg/constants"
	"github.com/dromara/carbon/v2"
)

// GetDateStartAndEndMill 获取日期开始结束时间戳（毫秒）
func GetDateStartAndEndMill(timeSearchType constants.TimeSearchType, startTime, endTime int64) (int64, int64) {
	// 获取当前时间并设置周起始日为周一
	now := carbon.Now().SetWeekStartsAt(carbon.Monday)

	// 转换为毫秒的辅助函数
	toMillis := func(t *carbon.Carbon) int64 {
		return t.Timestamp() * 1000
	}

	switch timeSearchType {
	case constants.TimeSearchTypeAll: // 所有时间
		return 0, toMillis(now.EndOfDay())

	case constants.TimeSearchTypeToday: // 当日
		return toMillis(now.StartOfDay()), toMillis(now.EndOfDay())

	case constants.TimeSearchTypeYesterday: // 昨日
		// 使用 SubDay 替代 Yesterday
		yesterday := now.SubDay()
		return toMillis(yesterday.StartOfDay()), toMillis(yesterday.EndOfDay())

	case constants.TimeSearchTypeThisWeek: // 本周
		return toMillis(now.StartOfWeek()), toMillis(now.EndOfWeek())

	case constants.TimeSearchTypeLastWeek: // 上周
		lastWeek := now.SubWeek()
		return toMillis(lastWeek.StartOfWeek()), toMillis(lastWeek.EndOfWeek())

	case constants.TimeSearchTypeThisMonth: // 本月
		return toMillis(now.StartOfMonth()), toMillis(now.EndOfMonth())

	case constants.TimeSearchTypeLastMonth: // 上月
		lastMonth := now.SubMonth()
		return toMillis(lastMonth.StartOfMonth()), toMillis(lastMonth.EndOfMonth())

	case constants.TimeSearchTypeCustom: // 自定义时间
		// 将秒级时间戳转换为Carbon对象
		startDate := carbon.CreateFromTimestamp(startTime)
		endDate := carbon.CreateFromTimestamp(endTime)
		return toMillis(startDate.StartOfDay()), toMillis(endDate.EndOfDay())

	default: // 默认返回当日开始，结束时间戳
		return toMillis(now.StartOfDay()), toMillis(now.EndOfDay())
	}
}

// GetDateStartAndEndSecond 获取日期开始结束时间戳（秒）
func GetDateStartAndEndSecond(timeSearchType constants.TimeSearchType, startTime, endTime int64) (int64, int64) {
	// 获取当前时间并设置周起始日为周一
	now := carbon.Now().SetWeekStartsAt(carbon.Monday)

	// 直接返回秒级时间戳
	toSeconds := func(t *carbon.Carbon) int64 {
		return t.Timestamp()
	}

	switch timeSearchType {
	case constants.TimeSearchTypeAll: // 所有时间
		return 0, toSeconds(now.EndOfDay())

	case constants.TimeSearchTypeToday: // 当日
		return toSeconds(now.StartOfDay()), toSeconds(now.EndOfDay())

	case constants.TimeSearchTypeYesterday: // 昨日
		yesterday := now.SubDay()
		return toSeconds(yesterday.StartOfDay()), toSeconds(yesterday.EndOfDay())

	case constants.TimeSearchTypeThisWeek: // 本周
		return toSeconds(now.StartOfWeek()), toSeconds(now.EndOfWeek())

	case constants.TimeSearchTypeLastWeek: // 上周
		lastWeek := now.SubWeek()
		return toSeconds(lastWeek.StartOfWeek()), toSeconds(lastWeek.EndOfWeek())

	case constants.TimeSearchTypeThisMonth: // 本月
		return toSeconds(now.StartOfMonth()), toSeconds(now.EndOfMonth())

	case constants.TimeSearchTypeLastMonth: // 上月
		lastMonth := now.SubMonth()
		return toSeconds(lastMonth.StartOfMonth()), toSeconds(lastMonth.EndOfMonth())

	case constants.TimeSearchTypeCustom: // 自定义时间
		// 将秒级时间戳转换为Carbon对象
		startDate := carbon.CreateFromTimestamp(startTime)
		endDate := carbon.CreateFromTimestamp(endTime)
		return toSeconds(startDate.StartOfDay()), toSeconds(endDate.EndOfDay())

	default: // 默认返回当日开始，结束时间戳
		return toSeconds(now.StartOfDay()), toSeconds(now.EndOfDay())
	}
}
