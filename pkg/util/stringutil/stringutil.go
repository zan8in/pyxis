package stringutil

import (
	"strings"
	"unicode/utf8"

	"github.com/axgle/mahonia"
)

// 字符串转 utf 8
func Str2UTF8(str string) string {
	if len(str) == 0 {
		return ""
	}
	if !utf8.ValidString(str) {
		return mahonia.NewDecoder("gb18030").ConvertString(str)
	}
	return str
}

// EqualFoldAny returns true if s is equal to any specified substring
func EqualFoldAny(s string, ss ...string) bool {
	for _, sss := range ss {
		if strings.EqualFold(s, sss) {
			return true
		}
	}
	return false
}
