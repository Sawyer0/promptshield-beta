package scanner

import (
	"unicode"
)

// LuhnCheck validates a candidate string using the Luhn algorithm.
// It extracts digits from the input and returns true if they form
// a plausible PAN (length >= 13) and pass checksum.
func LuhnCheck(s string) bool {
	digits := make([]int, 0, len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0
}

// SSNAreaValid returns true if the SSN area (first three digits) is in a valid range.
// This is a lightweight heuristic only; it does not guarantee validity.
func SSNAreaValid(s string) bool {
	n := 0
	area := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			area = area*10 + int(r-'0')
			n++
			if n == 3 {
				break
			}
		}
	}
	if n < 3 {
		return false
	}
	// 000, 666, and 900-999 are invalid historically
	if area == 0 || area == 666 || area >= 900 {
		return false
	}
	return true
}
