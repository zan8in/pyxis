package retryhttpclient

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

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

func simpleRtryHttpGet(target string) ([]byte, int, error) {
	if len(target) == 0 {
		return []byte(""), 0, errors.New("no target specified")
	}

	req, err := retryablehttp.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, err
	}

	// req.Header.Add("User-Agent", utils.RandomUA())

	resp, err := RtryNoRedirect.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return []byte(""), 0, err
	}

	reader := io.LimitReader(resp.Body, maxDefaultBody)
	respBody, err := io.ReadAll(reader)
	if err != nil {
		resp.Body.Close()
		return []byte(""), 0, err
	}

	return respBody, resp.StatusCode, err
}

// body is parameters 1
// headers is parameters 2
// statusCode is parameters 3
// err is parameters 4
func simpleRtryRedirectGet(target string) ([]byte, map[string][]string, int, error) {
	if len(target) == 0 {
		return []byte(""), nil, 0, errors.New("no target specified")
	}

	req, err := retryablehttp.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, 0, err
	}

	// req.Header.Add("User-Agent", utils.RandomUA())

	resp, err := RtryRedirect.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return []byte(""), nil, 0, err
	}

	reader := io.LimitReader(resp.Body, maxDefaultBody)
	respBody, err := io.ReadAll(reader)
	if err != nil {
		resp.Body.Close()
		return []byte(""), nil, 0, err
	}

	newheader := make(map[string][]string)
	for k := range resp.Header {
		newheader[k] = []string{resp.Header.Get(k)}

	}

	return respBody, newheader, resp.StatusCode, nil
}

// Check http or https And Check host live status
// returns response body and status code
// status code = -1 means server responded failed
func CheckHttpsAndLives(target string) (string, int) {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		_, statusCode, err := simpleRtryHttpGet(target)
		if err == nil {
			return target, statusCode
		}
		return target, -1
	}

	u, err := url.Parse("http://" + target)
	if err != nil {
		return target, -1
	}

	port := u.Port()

	switch {
	case port == "80" || len(port) == 0:
		_, statusCode, err := simpleRtryHttpGet("http://" + target)
		if err == nil {
			return "http://" + target, statusCode
		}
		return target, -1

	case port == "443" || strings.HasSuffix(port, "443"):
		_, statusCode, err := simpleRtryHttpGet("https://" + target)
		if err == nil {
			return "https://" + target, statusCode
		}
		return target, -1
	}

	resp, statusCode, err := simpleRtryHttpGet("http://" + target)
	if err == nil {
		if bytes.Contains(resp, []byte("<title>400 The plain HTTP request was sent to HTTPS port</title>")) {
			return "https://" + target, statusCode
		}
		return "http://" + target, statusCode
	}

	_, statusCode, err = simpleRtryHttpGet("https://" + target)
	if err == nil {
		return "https://" + target, statusCode
	}

	return target, -1
}

// Reverse URL Get request
func ReverseGet(target string) ([]byte, error) {
	if len(target) == 0 {
		return []byte(""), errors.New("target not find")
	}
	respBody, _, err := simpleRtryHttpGet(target)
	return respBody, err
}

func FingerPrintGet(target string) ([]byte, map[string][]string, int, error) {
	return simpleRtryRedirectGet(target)
}
