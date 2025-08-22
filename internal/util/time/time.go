package time

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatDuration formats a duration in human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// ParseDuration parses a duration string with extended format support
func ParseDuration(s string) (time.Duration, error) {
	// Try standard Go duration format first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	
	// Extended format support
	s = strings.ToLower(strings.TrimSpace(s))
	
	var total time.Duration
	var current strings.Builder
	
	for i, r := range s {
		if r >= '0' && r <= '9' || r == '.' {
			current.WriteRune(r)
		} else {
			if current.Len() == 0 {
				return 0, fmt.Errorf("invalid duration format: %s", s)
			}
			
			valueStr := current.String()
			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", valueStr)
			}
			
			// Find unit
			unit := s[i:]
			var duration time.Duration
			
			switch {
			case strings.HasPrefix(unit, "ns") || strings.HasPrefix(unit, "nanosecond"):
				duration = time.Duration(value)
			case strings.HasPrefix(unit, "us") || strings.HasPrefix(unit, "μs") || strings.HasPrefix(unit, "microsecond"):
				duration = time.Duration(value * float64(time.Microsecond))
			case strings.HasPrefix(unit, "ms") || strings.HasPrefix(unit, "millisecond"):
				duration = time.Duration(value * float64(time.Millisecond))
			case strings.HasPrefix(unit, "s") || strings.HasPrefix(unit, "second"):
				duration = time.Duration(value * float64(time.Second))
			case strings.HasPrefix(unit, "m") || strings.HasPrefix(unit, "minute"):
				duration = time.Duration(value * float64(time.Minute))
			case strings.HasPrefix(unit, "h") || strings.HasPrefix(unit, "hour"):
				duration = time.Duration(value * float64(time.Hour))
			case strings.HasPrefix(unit, "d") || strings.HasPrefix(unit, "day"):
				duration = time.Duration(value * float64(24*time.Hour))
			case strings.HasPrefix(unit, "w") || strings.HasPrefix(unit, "week"):
				duration = time.Duration(value * float64(7*24*time.Hour))
			default:
				return 0, fmt.Errorf("unknown time unit: %s", unit)
			}
			
			total += duration
			current.Reset()
			
			// Skip the unit
			for j := i; j < len(s); j++ {
				if s[j] >= '0' && s[j] <= '9' {
					s = s[j:]
					break
				}
				if j == len(s)-1 {
					return total, nil
				}
			}
			i = -1 // Reset loop
		}
	}
	
	return total, nil
}

// Now returns current time
func Now() time.Time {
	return time.Now()
}

// NowUTC returns current time in UTC
func NowUTC() time.Time {
	return time.Now().UTC()
}

// Unix returns time from Unix timestamp
func Unix(sec int64) time.Time {
	return time.Unix(sec, 0)
}

// UnixMilli returns time from Unix timestamp in milliseconds
func UnixMilli(msec int64) time.Time {
	return time.Unix(msec/1000, (msec%1000)*1000000)
}

// UnixMicro returns time from Unix timestamp in microseconds
func UnixMicro(usec int64) time.Time {
	return time.Unix(usec/1000000, (usec%1000000)*1000)
}

// ToUnix converts time to Unix timestamp
func ToUnix(t time.Time) int64 {
	return t.Unix()
}

// ToUnixMilli converts time to Unix timestamp in milliseconds
func ToUnixMilli(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

// ToUnixMicro converts time to Unix timestamp in microseconds
func ToUnixMicro(t time.Time) int64 {
	return t.UnixNano() / 1000
}

// ToUnixNano converts time to Unix timestamp in nanoseconds
func ToUnixNano(t time.Time) int64 {
	return t.UnixNano()
}

// StartOfDay returns the start of the day (00:00:00)
func StartOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day (23:59:59.999999999)
func EndOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek returns the start of the week (Monday 00:00:00)
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	days := weekday - 1
	return StartOfDay(t.AddDate(0, 0, -days))
}

// EndOfWeek returns the end of the week (Sunday 23:59:59.999999999)
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth returns the start of the month (1st day 00:00:00)
func StartOfMonth(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the end of the month (last day 23:59:59.999999999)
func EndOfMonth(t time.Time) time.Time {
	return EndOfDay(StartOfMonth(t).AddDate(0, 1, -1))
}

// StartOfYear returns the start of the year (January 1st 00:00:00)
func StartOfYear(t time.Time) time.Time {
	year, _, _ := t.Date()
	return time.Date(year, 1, 1, 0, 0, 0, 0, t.Location())
}

// EndOfYear returns the end of the year (December 31st 23:59:59.999999999)
func EndOfYear(t time.Time) time.Time {
	year, _, _ := t.Date()
	return time.Date(year, 12, 31, 23, 59, 59, 999999999, t.Location())
}

// Age calculates age in years from a birth date
func Age(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	
	// Adjust if birthday hasn't occurred this year
	if now.Month() < birthDate.Month() || 
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		age--
	}
	
	return age
}

// DaysUntil calculates days until a future date
func DaysUntil(future time.Time) int {
	now := time.Now()
	diff := future.Sub(now)
	return int(diff.Hours() / 24)
}

// DaysSince calculates days since a past date
func DaysSince(past time.Time) int {
	now := time.Now()
	diff := now.Sub(past)
	return int(diff.Hours() / 24)
}

// IsLeapYear checks if a year is a leap year
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// DaysInMonth returns the number of days in a month
func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// IsWeekend checks if a date is a weekend (Saturday or Sunday)
func IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// IsWeekday checks if a date is a weekday (Monday to Friday)
func IsWeekday(t time.Time) bool {
	return !IsWeekend(t)
}

// NextWeekday returns the next occurrence of a specific weekday
func NextWeekday(t time.Time, weekday time.Weekday) time.Time {
	days := int(weekday) - int(t.Weekday())
	if days <= 0 {
		days += 7
	}
	return t.AddDate(0, 0, days)
}

// PreviousWeekday returns the previous occurrence of a specific weekday
func PreviousWeekday(t time.Time, weekday time.Weekday) time.Time {
	days := int(t.Weekday()) - int(weekday)
	if days <= 0 {
		days += 7
	}
	return t.AddDate(0, 0, -days)
}

// Sleep pauses execution for a duration
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// Timeout creates a timeout channel
func Timeout(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Ticker creates a ticker channel
func Ticker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

// Timer creates a timer
func Timer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}

// Elapsed measures elapsed time since start
func Elapsed(start time.Time) time.Duration {
	return time.Since(start)
}

// ElapsedSince is an alias for Elapsed
func ElapsedSince(start time.Time) time.Duration {
	return Elapsed(start)
}

// Measure measures execution time of a function
func Measure(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// MeasureFunc measures execution time and returns result
func MeasureFunc(fn func() interface{}) (interface{}, time.Duration) {
	start := time.Now()
	result := fn()
	return result, time.Since(start)
}

// Stopwatch provides a simple stopwatch functionality
type Stopwatch struct {
	start   time.Time
	elapsed time.Duration
	running bool
}

// NewStopwatch creates a new stopwatch
func NewStopwatch() *Stopwatch {
	return &Stopwatch{}
}

// Start starts the stopwatch
func (s *Stopwatch) Start() {
	if !s.running {
		s.start = time.Now()
		s.running = true
	}
}

// Stop stops the stopwatch
func (s *Stopwatch) Stop() {
	if s.running {
		s.elapsed += time.Since(s.start)
		s.running = false
	}
}

// Reset resets the stopwatch
func (s *Stopwatch) Reset() {
	s.elapsed = 0
	s.running = false
}

// Restart resets and starts the stopwatch
func (s *Stopwatch) Restart() {
	s.Reset()
	s.Start()
}

// Elapsed returns the elapsed time
func (s *Stopwatch) Elapsed() time.Duration {
	if s.running {
		return s.elapsed + time.Since(s.start)
	}
	return s.elapsed
}

// IsRunning returns whether the stopwatch is running
func (s *Stopwatch) IsRunning() bool {
	return s.running
}

// ParseCommonFormats attempts to parse time from common formats
func ParseCommonFormats(timeStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000000000",
		"2006-01-02",
		"15:04:05",
		"01/02/2006",
		"01/02/2006 15:04:05",
		"02/01/2006",
		"02/01/2006 15:04:05",
		"2006/01/02",
		"2006/01/02 15:04:05",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// FormatRelative formats time relative to now (e.g., "2 hours ago", "in 3 days")
func FormatRelative(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < 0 {
		diff = -diff
		return formatRelativeFuture(diff)
	}
	
	return formatRelativePast(diff)
}

func formatRelativePast(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if d < 30*24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if d < 365*24*time.Hour {
		months := int(d.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	
	years := int(d.Hours() / (24 * 365))
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

func formatRelativeFuture(d time.Duration) string {
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "in 1 minute"
		}
		return fmt.Sprintf("in %d minutes", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "in 1 hour"
		}
		return fmt.Sprintf("in %d hours", hours)
	}
	if d < 30*24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "in 1 day"
		}
		return fmt.Sprintf("in %d days", days)
	}
	if d < 365*24*time.Hour {
		months := int(d.Hours() / (24 * 30))
		if months == 1 {
			return "in 1 month"
		}
		return fmt.Sprintf("in %d months", months)
	}
	
	years := int(d.Hours() / (24 * 365))
	if years == 1 {
		return "in 1 year"
	}
	return fmt.Sprintf("in %d years", years)
}

// ObserveDuration records a duration to a Prometheus observer (histogram/summary)
func ObserveDuration(observer interface{ Observe(float64) }, duration time.Duration) {
	observer.Observe(duration.Seconds())
}