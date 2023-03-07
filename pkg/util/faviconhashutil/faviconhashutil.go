package faviconhashutil

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/spaolacci/murmur3"
	"github.com/zan8in/retryablehttp/pkg/utils/stringsutil"
)

func isContentTypeImage(data []byte) bool {
	contentType := http.DetectContentType(data)
	return stringsutil.HasPrefixAny(contentType, "image/")
}

func murmurhash(data []byte) int32 {
	stdBase64 := base64.StdEncoding.EncodeToString(data)
	stdBase64 = InsertInto(stdBase64, 76, '\n')
	hasher := murmur3.New32WithSeed(0)
	hasher.Write([]byte(stdBase64))
	return int32(hasher.Sum32())
}

func FaviconHash(data []byte) (int32, error) {
	if isContentTypeImage(data) {
		return murmurhash(data), nil
	}

	return 0, errors.New("content type is not image")
}

func InsertInto(s string, interval int, sep rune) string {
	var buffer bytes.Buffer
	before := interval - 1
	last := len(s) - 1
	for i, char := range s {
		buffer.WriteRune(char)
		if i%interval == before && i != last {
			buffer.WriteRune(sep)
		}
	}
	buffer.WriteRune(sep)
	return buffer.String()
}

// Base64 returns base64 of given byte array
func Base64(bin []byte) string {
	return base64.StdEncoding.EncodeToString(bin)
}
