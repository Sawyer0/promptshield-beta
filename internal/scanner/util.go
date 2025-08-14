package scanner

import "bytes"

func bytesIndexByte(b []byte, c byte) int {
    return bytes.IndexByte(b, c)
}

func isWordChar(r rune) bool {
	if r >= '0' && r <= '9' {
		return true
	}
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	return r == '_' || r == '-'
}
