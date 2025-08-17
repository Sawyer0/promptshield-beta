package scanner

import (
	"strings"
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

// IBANValid validates an International Bank Account Number using the standard mod-97 check.
// It performs a generic check without enforcing per-country exact lengths, but requires total
// length to be between 15 and 34 characters inclusive, consisting of letters and digits.
// Steps:
// 1) Normalize: remove spaces, upper-case
// 2) Move first 4 chars to the end
// 3) Replace letters A..Z with 10..35
// 4) Compute number mod 97, must equal 1
func IBANValid(s string) bool {
	// Normalize
	n := make([]rune, 0, len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '-' {
			continue
		}
		if unicode.IsLetter(r) {
			r = unicode.ToUpper(r)
		}
		n = append(n, r)
	}
	if len(n) < 15 || len(n) > 34 {
		return false
	}
	// All letters/digits only
	for _, r := range n {
		if !(unicode.IsDigit(r) || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	// Move first 4 to end
	if len(n) < 4 {
		return false
	}
	rearr := append(n[4:], n[:4]...)
	// Convert to mod-97 iteratively to avoid big ints
	mod := 0
	for _, r := range rearr {
		var chunk string
		if unicode.IsDigit(r) {
			chunk = string(r)
		} else {
			// A=10, B=11, ..., Z=35
			v := int(r-'A') + 10
			if v < 10 || v > 35 {
				return false
			}
			chunk = itoa2(v) // at most two digits
		}
		for i := 0; i < len(chunk); i++ {
			mod = (mod*10 + int(chunk[i]-'0')) % 97
		}
	}
	return mod == 1
}

// EmailValid performs a lightweight syntax check for an email address candidate.
// It rejects obvious false positives while avoiding heavy/strict RFC validation.
// Rules:
// - exactly one '@'
// - local and domain non-empty; local <= 64, total <= 254
// - domain has at least one dot and labels are alnum/hyphen, not starting/ending with '-'
func EmailValid(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 || len(s) > 254 {
		return false
	}
	at := strings.Count(s, "@")
	if at != 1 {
		return false
	}
	parts := strings.SplitN(s, "@", 2)
	local, domain := parts[0], parts[1]
	if len(local) == 0 || len(local) > 64 || len(domain) == 0 {
		return false
	}
	// Basic domain check
	if !strings.Contains(domain, ".") {
		return false
	}
	labels := strings.Split(domain, ".")
	for _, lab := range labels {
		if len(lab) == 0 || len(lab) > 63 {
			return false
		}
		if lab[0] == '-' || lab[len(lab)-1] == '-' {
			return false
		}
		for i := 0; i < len(lab); i++ {
			c := lab[i]
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}
	return true
}

// itoa2 converts an integer in [10,35] to its two-digit decimal string
func itoa2(v int) string {
	// v is guaranteed 10..35
	tens := byte('0' + (v / 10))
	ones := byte('0' + (v % 10))
	return string([]byte{tens, ones})
}
