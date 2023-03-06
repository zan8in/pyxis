package stringutil

import (
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
