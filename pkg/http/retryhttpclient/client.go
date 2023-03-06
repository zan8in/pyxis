package retryhttpclient

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"regexp"
	"runtime"
	"time"

	"github.com/zan8in/pyxis/pkg/result"
	"github.com/zan8in/pyxis/pkg/util/randutil"
	"github.com/zan8in/pyxis/pkg/util/stringutil"
	"github.com/zan8in/retryablehttp"
	"golang.org/x/net/context"
	"golang.org/x/net/proxy"
)

var (
	RtryRedirect   *retryablehttp.Client
	RtryNoRedirect *retryablehttp.Client

	RtryNoRedirectHttpClient *http.Client
	RtryRedirectHttpClient   *http.Client
	defaultMaxRedirects      = 10
)

const maxDefaultBody = 2 * 1024 * 1024

type Options struct {
	Timeout int
	Retries int

	Proxy string
}

func Init(options *Options) (err error) {
	retryableHttpOptions := retryablehttp.DefaultOptionsSpraying
	maxIdleConns := 0
	maxConnsPerHost := 0
	maxIdleConnsPerHost := -1
	disableKeepAlives := true // 默认 false

	// retryableHttpOptions = retryablehttp.DefaultOptionsSingle
	// disableKeepAlives = false
	// maxIdleConnsPerHost = 500
	// maxConnsPerHost = 500

	maxIdleConns = 1000                        //
	maxIdleConnsPerHost = runtime.NumCPU() * 2 //
	idleConnTimeout := 15 * time.Second        //
	tLSHandshakeTimeout := 5 * time.Second     //

	dialer := &net.Dialer{ //
		Timeout:   time.Duration(options.Timeout) * time.Second,
		KeepAlive: 15 * time.Second,
	}

	retryableHttpOptions.RetryWaitMax = 10 * time.Second
	retryableHttpOptions.RetryMax = options.Retries

	tlsConfig := &tls.Config{
		Renegotiation:      tls.RenegotiateOnceAsClient,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
	}

	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		MaxConnsPerHost:     maxConnsPerHost,
		TLSClientConfig:     tlsConfig,
		DisableKeepAlives:   disableKeepAlives,
		TLSHandshakeTimeout: tLSHandshakeTimeout, //
		IdleConnTimeout:     idleConnTimeout,     //
	}

	// transport = &http.Transport{
	// 	// DialContext:         dialer.Dial,
	// 	// DialTLSContext:      dialer.DialTLS,
	// 	MaxIdleConns:        500,
	// 	MaxIdleConnsPerHost: 500,
	// 	MaxConnsPerHost:     500,
	// 	TLSClientConfig:     tlsConfig,
	// }

	// proxy

	if ProxyURL != "" {
		if proxyURL, err := url.Parse(ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	} else if ProxySocksURL != "" {
		socksURL, proxyErr := url.Parse(ProxySocksURL)
		if proxyErr != nil {
			return proxyErr
		}
		dialer, err := proxy.FromURL(socksURL, proxy.Direct)
		if err != nil {
			return err
		}

		dc := dialer.(interface {
			DialContext(ctx context.Context, network, addr string) (net.Conn, error)
		})
		if proxyErr == nil {
			transport.DialContext = dc.DialContext
			transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				// upgrade proxy connection to tls
				conn, err := dc.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				return tls.Client(conn, tlsConfig), nil
			}
		}
	}

	// follow redirects client
	// clientCookieJar, _ := cookiejar.New(nil)

	httpRedirectClient := http.Client{
		Transport: transport,
		Timeout:   time.Duration(options.Timeout) * time.Second,
		// Jar:       clientCookieJar,
	}

	RtryRedirect = retryablehttp.NewWithHTTPClient(&httpRedirectClient, retryableHttpOptions)
	RtryRedirect.CheckRetry = retryablehttp.HostSprayRetryPolicy()
	RtryRedirectHttpClient = RtryRedirect.HTTPClient

	// whitespace

	// disabled follow redirects client
	// clientNoRedirectCookieJar, _ := cookiejar.New(nil)

	httpNoRedirectClient := http.Client{
		Transport: transport,
		Timeout:   time.Duration(options.Timeout) * time.Second,
		// Jar:           clientNoRedirectCookieJar,
		CheckRedirect: makeCheckRedirectFunc(false, defaultMaxRedirects),
	}

	RtryNoRedirect = retryablehttp.NewWithHTTPClient(&httpNoRedirectClient, retryableHttpOptions)
	RtryNoRedirect.CheckRetry = retryablehttp.HostSprayRetryPolicy()
	RtryNoRedirectHttpClient = RtryNoRedirect.HTTPClient

	return err
}

type checkRedirectFunc func(req *http.Request, via []*http.Request) error

func makeCheckRedirectFunc(followRedirects bool, maxRedirects int) checkRedirectFunc {
	return func(req *http.Request, via []*http.Request) error {
		if !followRedirects {
			return http.ErrUseLastResponse
		}

		if maxRedirects == 0 {
			if len(via) > defaultMaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		}

		if len(via) > maxRedirects {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

func GetHttpRequest(target string) (result.HostResult, error) {
	var (
		err    error
		result result.HostResult
	)

	req, err := retryablehttp.NewRequest(http.MethodGet, target, nil)
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

	resp, err := RtryRedirect.Do(req)
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

	utf8RespBody := stringutil.Str2UTF8(string(respBody))

	result.FullUrl = target
	result.StatusCode = resp.StatusCode
	result.Title = getTitle(utf8RespBody)
	result.ResponseTime = milliseconds
	result.ContentLength = resp.ContentLength
	result.Body = utf8RespBody

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
