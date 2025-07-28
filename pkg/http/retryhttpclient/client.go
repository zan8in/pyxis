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

	result.FullUrl = target
	result.StatusCode = resp.StatusCode
	result.Title = getTitle(stringutil.Str2UTF8(string(respBody)))
	result.ResponseTime = milliseconds
	result.Body = string(respBody)
	result.ContentLength = int64(len(respBody))

	// fingerprint
	newRespHeader := make(map[string]string)
	rawHeaderBuilder := strings.Builder{}
	for k := range resp.Header {
		newRespHeader[strings.ToLower(k)] = resp.Header.Get(k)

		rawHeaderBuilder.WriteString(k)
		rawHeaderBuilder.WriteString(": ")
		rawHeaderBuilder.WriteString(resp.Header.Get(k))
		rawHeaderBuilder.WriteString("\n")
	}
	result.Headers = newRespHeader
	result.RawBody = []byte(stringutil.Str2UTF8(string(respBody)))
	result.RawHeader = []byte(strings.Trim(rawHeaderBuilder.String(), "\n"))
	result.Raw = []byte(resp.Proto + " " + resp.Status + "\n" + strings.Trim(rawHeaderBuilder.String(), "\n") + "\n\n" + stringutil.Str2UTF8(string(respBody)))

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
