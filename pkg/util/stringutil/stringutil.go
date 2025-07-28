package stringutil

import (
	"strings"
	"unicode/utf8"

	"github.com/axgle/mahonia"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding"
)

// 常见编码列表
var encodings = []struct {
	name string
	enc  encoding.Encoding
}{
	{"GB18030", simplifiedchinese.GB18030},
	{"GBK", simplifiedchinese.GBK},
	{"GB2312", simplifiedchinese.HZGB2312},
	{"BIG5", traditionalchinese.Big5},
	{"EUC-JP", japanese.EUCJP},
	{"SHIFT_JIS", japanese.ShiftJIS},
	{"EUC-KR", korean.EUCKR},
	{"ISO-8859-1", charmap.ISO8859_1},
	{"ISO-8859-2", charmap.ISO8859_2},
	{"windows-1251", charmap.Windows1251},
	{"windows-1252", charmap.Windows1252},
}

// 字符串转 utf 8
func Str2UTF8(str string) string {
	if len(str) == 0 {
		return ""
	}
	
	// 如果已经是有效的UTF-8，直接返回
	if utf8.ValidString(str) {
		return str
	}
	
	// 尝试使用mahonia转换（保留原有逻辑作为首选）
	result := mahonia.NewDecoder("gb18030").ConvertString(str)
	if utf8.ValidString(result) {
		return result
	}
	
	// 尝试其他编码
	for _, enc := range encodings {
		if decoder := enc.enc.NewDecoder(); decoder != nil {
			result, err := decoder.String(str)
			if err == nil && utf8.ValidString(result) {
				return result
			}
		}
	}
	
	// 如果所有尝试都失败，返回原始字符串
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
