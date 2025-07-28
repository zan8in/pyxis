package retryhttpclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
	"time"

	"github.com/zan8in/pyxis/pkg/result"
	"github.com/zan8in/pyxis/pkg/util/randutil"
	"github.com/zan8in/pyxis/pkg/util/stringutil"
	"github.com/zan8in/retryablehttp"
)

var (
	RedirectClient *retryablehttp.Client
)

const maxDefaultBody = 2 * 1024 * 1024

type Options struct {
	Timeout int
	Retries int
	Proxy   string
}

func Init(options *Options) (err error) {
	po := &retryablehttp.DefaultPoolOptions
	po.Proxy = options.Proxy
	po.Timeout = options.Timeout
	po.Retries = options.Retries
	po.EnableRedirect(retryablehttp.FollowAllRedirect)

	retryablehttp.InitClientPool(po)

	if RedirectClient, err = retryablehttp.GetPool(po); err != nil {
		return err
	}

	return nil
}

func Get(target string) (result.HostResult, error) {
	var (
		err    error
		result result.HostResult
	)

	timeoutDuration := time.Duration(RedirectClient.HTTPClient.Timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return result, err
	}

	req.Header.Add("User-Agent", randutil.RandomUA())

	// latency
	var milliseconds int64
	start := time.Now()
	trace := httptrace.ClientTrace{}
	trace.GotFirstResponseByte = func() {
		milliseconds = time.Since(start).Nanoseconds() / 1e6
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &trace))

	resp, err := RedirectClient.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return result, err
	}
	defer resp.Body.Close()

	reader := io.LimitReader(resp.Body, maxDefaultBody)
	respBody, err := io.ReadAll(reader)
	if err != nil {
		return result, err
	}

	// 设置基本字段
	result.FullUrl = target
	result.StatusCode = resp.StatusCode
	result.ResponseTime = milliseconds
	result.ContentLength = int64(len(respBody))

	// 处理响应体，避免多次转换
	utf8Body := stringutil.Str2UTF8(string(respBody))
	result.Body = utf8Body
	result.Title = getTitle(utf8Body)
	result.RawBody = []byte(utf8Body)

	// 处理响应头
	newRespHeader := make(map[string]string, len(resp.Header))
	rawHeaderBuilder := strings.Builder{}
	rawHeaderBuilder.Grow(1024) // 预分配空间

	for k := range resp.Header {
		newRespHeader[strings.ToLower(k)] = resp.Header.Get(k)
		rawHeaderBuilder.WriteString(k)
		rawHeaderBuilder.WriteString(": ")
		rawHeaderBuilder.WriteString(resp.Header.Get(k))
		rawHeaderBuilder.WriteString("\n")
	}
	result.Headers = newRespHeader

	// 构建原始响应数据
	rawHeader := strings.TrimSuffix(rawHeaderBuilder.String(), "\n")
	result.RawHeader = []byte(rawHeader)

	// 构建完整的原始响应
	rawBuilder := strings.Builder{}
	rawBuilder.Grow(len(resp.Proto) + len(resp.Status) + len(rawHeader) + len(utf8Body) + 10)
	rawBuilder.WriteString(resp.Proto)
	rawBuilder.WriteString(" ")
	rawBuilder.WriteString(resp.Status)
	rawBuilder.WriteString("\n")
	rawBuilder.WriteString(rawHeader)
	rawBuilder.WriteString("\n\n")
	rawBuilder.WriteString(utf8Body)
	result.Raw = []byte(rawBuilder.String())

	return result, nil
}

var RegexTitle = regexp.MustCompile(`(?i:)<title>(.*?)</title>`)

func getTitle(body string) string {
	titleSlice := RegexTitle.FindStringSubmatch(body)
	if len(titleSlice) == 2 {
		return titleSlice[1]
	}
	return ""
}
