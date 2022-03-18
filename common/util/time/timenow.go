package time

import (
	"time"

	"github.com/jinzhu/now"
)

const (
	Minute = 60
	Hour   = 3600
	Day    = 86400
)

// Now: 获取当前时间
func Now() int64 {
	return time.Now().Unix()
}

// NowMinute: 获取当前分钟
func NowMinute() int64 {
	return int64(time.Now().Minute())
}

// Week: 获取当前是第几周
func Week(tm int64) int64 {
	if tm == 0 {
		_, week := time.Now().ISOWeek()
		return int64(week)
	} else {
		t := time.Unix(tm, 0)
		_, week := t.ISOWeek()
		return int64(week)
	}
}

// Weekday: 获取星期几
func Weekday(tm int64) int64 {
	if tm == 0 {
		return int64(time.Now().Weekday())
	} else {
		t := time.Unix(tm, 0)
		return int64(t.Weekday())
	}
}

// Today: 获取今天日期字符串
func Today() string {
	return time.Now().Format("2006-01-02")
}

// TodayTime: 获取当前时间字符串
func TodayTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func DateFormat(tm int64) string {
	return time.Unix(tm, 0).Format("2006-01-02")
}

func MonthFormat(tm int64) string {
	return time.Unix(tm, 0).Format("2006-01")
}

func Format(tm int64) string {
	return time.Unix(tm, 0).Format("2006-01-02 15:04:05")
}

func TodayStart() int64 {
	return now.BeginningOfDay().Unix()
}

func TodayEnd() int64 {
	return now.EndOfDay().Unix()
}

func DateParse(date string) (timestamp int64) {
	t, err := now.Parse(date)
	if err != nil {
		return
	}
	return t.Unix()
}

func DateBeginTimestamp(date string) (timestamp int64) {
	t, err := now.Parse(date)
	if err != nil {
		return
	}
	return now.With(t).BeginningOfDay().Unix()
}

func DateEndTimestamp(date string) (timestamp int64) {
	t, err := now.Parse(date)
	if err != nil {
		return
	}
	return now.With(t).EndOfDay().Unix()
}

// t: timestamp
// get begin timestamp of day DayBeginTimestamp
func DayBeginTimestamp(tm int64) (timestamp int64) {
	t := time.Unix(tm, 0)
	return now.With(t).BeginningOfDay().Unix()
}

// t: timestamp
// get end timestamp of day DayEndTimestamp
func DayEndTimestamp(tm int64) (timestamp int64) {
	t := time.Unix(tm, 0)
	return now.With(t).EndOfDay().Unix()
}

// t: timestamp
// get begin timestamp of week WeekBeginTimestamp
func WeekBeginTimestamp(tm int64) (timestamp int64) {
	t := time.Unix(tm, 0)
	return now.With(t).BeginningOfWeek().Unix()
}

// t: timestamp
// get end timestamp of week WeekEndTimestamp
func WeekEndTimestamp(tm int64) (timestamp int64) {
	t := time.Unix(tm, 0)
	return now.With(t).EndOfWeek().Unix()
}

func LastMonthBeginTimestamp() (timestamp int64) {
	year, month, _ := time.Now().Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, time.Local).AddDate(0, -1, 0).Unix()
}

// t: timestamp
// get begin timestamp of month MonthBeginTimestamp
func MonthBeginTimestamp(tm int64) (timestamp int64) {
	t := time.Unix(tm, 0)
	return now.With(t).BeginningOfMonth().Unix()
}

// t: timestamp
// get end timestamp of month MonthEndTimestamp
func MonthEndTimestamp(tm int64) (timestamp int64) {
	t := time.Unix(tm, 0)
	return now.With(t).EndOfMonth().Unix()
}
