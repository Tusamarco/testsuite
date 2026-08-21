package utils

import (
	"fmt"
	"strings"
	"time"
)

const (
	// DayFormat and TimeFormat translated to Go's reference time layout:
	// Mon Jan 2 15:04:05 -0700 MST 2006
	DayFormat  = "2006-01-02"
	TimeFormat = "15:04:05"
)

// GetDayFormat returns the default day format layout.
func GetDayFormat() string {
	return DayFormat
}

// GetTimeFormat returns the default time format layout.
func GetTimeFormat() string {
	return TimeFormat
}

// GetFullDate formats a Unix millisecond timestamp to DayFormat_TimeFormat.
func GetFullDate(systemTimeMilli int64) string {
	t := time.UnixMilli(systemTimeMilli)
	return t.Format(DayFormat + "_" + TimeFormat)
}

// GetCurrentDay returns the current date as a string.
func GetCurrentDay() string {
	return time.Now().Format(DayFormat)
}

// GetCurrentTime returns the current time as a string.
func GetCurrentTime() string {
	return time.Now().Format(TimeFormat)
}

// GetCurrentTimeFromTime returns the time formatted from a given time.Time object.
// (Replaces the overloaded Calendar version).
func GetCurrentTimeFromTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// GetCurrent returns the current date and time.
func GetCurrent() string {
	return time.Now().Format(DayFormat + " " + TimeFormat)
}

// GetCurrentFromTime returns formatted date and time from a given time.Time.
func GetCurrentFromTime(t time.Time) string {
	return t.Format(DayFormat + " " + TimeFormat)
}

// GetDate parses a date string based on a format.
// Note: This uses a helper to convert Java SimpleDateFormat tokens to Go layouts.
func GetDate(dateStr, javaFormatStr string) (time.Time, error) {
	goLayout := javaFormatToGoLayout(javaFormatStr)
	return time.Parse(goLayout, dateStr)
}

// GetYearFromTime returns the year from a given time.Time.
func GetYearFromTime(t time.Time) string {
	return fmt.Sprintf("%d", t.Year())
}

// GetMonthFromTime returns the month from a given time.Time.
func GetMonthFromTime(t time.Time) string {
	return t.Month().String()
}

// GetDayNumberFromTime returns the day of the month from a given time.Time.
func GetDayNumberFromTime(t time.Time) string {
	return fmt.Sprintf("%02d", t.Day())
}

// GetTime parses a date/time string using the default format.
// (Replaces the getCalendar method).
func GetTime(dateTime string) *time.Time {
	if dateTime == "" {
		return nil
	}
	t, err := time.Parse(DayFormat+" "+TimeFormat, dateTime)
	if err != nil {
		return nil
	}
	return &t
}

// AddDays adds a specific number of days to a given time.
// (Replaces getCalendarFromCalendarDateAddDays).
func AddDays(t time.Time, days int) time.Time {
	// Go's AddDate handles leap years and month rollovers automatically.
	return t.AddDate(0, 0, days)
}

// GetTimeStamp returns a timestamp formatted as "yyyy-MM-dd hh:mm:ss" (Java) -> "2006-01-02 03:04:05" (Go).
func GetTimeStamp(systemTimeMilli int64) string {
	if systemTimeMilli == 0 {
		return ""
	}
	return GetTimeStampWithFormat(systemTimeMilli, "yyyy-MM-dd hh:mm:ss")
}

// GetTimeStampWithFormat returns a timestamp using a custom Java-style format string.
func GetTimeStampWithFormat(systemTimeMilli int64, javaFormatStr string) string {
	if systemTimeMilli == 0 {
		return ""
	}
	t := time.UnixMilli(systemTimeMilli)
	return t.Format(javaFormatToGoLayout(javaFormatStr))
}

// GetTimeStampFromTime returns a timestamp using a given time.Time and format.
func GetTimeStampFromTime(t *time.Time, javaFormatStr string) string {
	if t == nil {
		return ""
	}
	if javaFormatStr == "" {
		return t.Format(DayFormat + " " + TimeFormat)
	}
	return t.Format(javaFormatToGoLayout(javaFormatStr))
}

// --- Helper Functions ---

// javaFormatToGoLayout is a helper function to convert common Java SimpleDateFormat
// tokens into Go's specific reference time layout.
func javaFormatToGoLayout(javaFormat string) string {
	r := strings.NewReplacer(
		"yyyy", "2006",
		"MM", "01",
		"dd", "02",
		"HH", "15", // 24-hour clock
		"hh", "03", // 12-hour clock
		"mm", "04",
		"ss", "05",
	)
	return r.Replace(javaFormat)
}
