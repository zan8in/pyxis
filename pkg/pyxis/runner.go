package pyxis

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/gologger"
	"github.com/zan8in/libra"
	"github.com/zan8in/pyxis/pkg/favicon"
	"github.com/zan8in/pyxis/pkg/http/retryhttpclient"
	"github.com/zan8in/pyxis/pkg/result"
	"github.com/zan8in/pyxis/pkg/util/iputil"
)

type Runner struct {
	Options *Options

	ticker *time.Ticker
	wgscan sizedwaitgroup.SizedWaitGroup

	hostChan chan string

	ResultChan chan *result.HostResult
	Result     *result.Result

	hostTempFile string

	Phase Phase
}

func NewRunner(options *Options) (*Runner, error) {
	var (
		err error
	)

	runner := &Runner{
		Options:    options,
		hostChan:   make(chan string),
		ResultChan: make(chan *result.HostResult),
		Result:     result.NewResult(),
	}

	if err = retryhttpclient.Init(&retryhttpclient.Options{
		Retries: options.Retries,
		Timeout: options.Timeout,
		Proxy:   options.Proxy,
	}); err != nil {
		return runner, err
	}

	runner.wgscan = sizedwaitgroup.New(options.RateLimit)
	runner.ticker = time.NewTicker(time.Second / time.Duration(options.RateLimit))

	return runner, err
}

func (r *Runner) Run() error {
	defer r.Close()

	go func() {
		if err := r.PreprocessHost(); err != nil {
			gologger.Fatal().Msg(err.Error())
		}
	}()

	go r.Listener()

	r.start()

	r.Delay()

	r.WriteOutput()

	return nil
}

func (r *Runner) ApiRun() error {
	defer r.Close()

	go func() {
		if err := r.PreprocessHost(); err != nil {
			gologger.Fatal().Msg(err.Error())
		}
	}()

	go r.ApiListener()

	r.start()

	r.Delay()

	return nil
}

func (r *Runner) Delay() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if r.Phase.Is(Done) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	wg.Wait()
}

func (r *Runner) Listener() {
	for result := range r.ResultChan {
		r.Result.SetHostResult(result.FullUrl, result)
		r.print(result)
	}
	r.Phase.Set(Done)
}

func (r *Runner) ApiListener() {
	for result := range r.ResultChan {
		r.Result.SetHostResult(result.FullUrl, result)
	}
	r.Phase.Set(Done)
}

func (r *Runner) start() {
	defer close(r.ResultChan)

	r.Phase.Set(Scan)

	for host := range r.hostChan {
		r.wgscan.Add()
		go func(host string) {
			if result, err := r.scanHost(host); err == nil {
				r.ResultChan <- &result
			}
		}(host)
	}
	r.wgscan.Wait()
}

func (r *Runner) scanHost(host string) (result.HostResult, error) {
	defer r.wgscan.Done()

	if len(strings.TrimSpace(host)) == 0 {
		return result.HostResult{}, fmt.Errorf("host %q is empty", host)
	}

	var (
		err       error
		result    result.HostResult
		parseHost string
		parsePort string
	)

	if strings.HasPrefix(host, HTTPS_PREFIX) {
		result, err = retryhttpclient.GetHttpRequest(host)
		if err != nil {
			return result, err
		}
		result.Port = 443
		result.TLS = true
		result.Host = ""
		u, err := url.Parse(host)
		if err == nil {
			result.Host = u.Hostname()
			result.IP = iputil.GetDomainIP(u.Hostname())
		}
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = getFingerprint(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), result.Headers)
		return result, nil
	}

	if strings.HasPrefix(host, HTTP_PREFIX) {
		result, err = retryhttpclient.GetHttpRequest(host)
		if err != nil {
			return result, err
		}
		result.Port = 80
		result.TLS = false
		result.Host = ""
		u, err := url.Parse(host)
		if err == nil {
			result.Host = u.Hostname()
			result.IP = iputil.GetDomainIP(u.Hostname())
		}
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = getFingerprint(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), result.Headers)
		return result, nil
	}

	u, err := url.Parse(HTTP_PREFIX + host)
	if err != nil {
		return result, err
	}
	parseHost = u.Hostname()
	parsePort = u.Port()

	switch {
	case parsePort == "80":
		result, err = retryhttpclient.GetHttpRequest(HTTP_PREFIX + host)
		if err != nil {
			return result, err
		}
		result.Port = 80
		result.TLS = false
		result.Host = parseHost
		result.IP = iputil.GetDomainIP(parseHost)
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = getFingerprint(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), result.Headers)
		return result, nil

	case parsePort == "443":
		result, err = retryhttpclient.GetHttpRequest(HTTPS_PREFIX + host)
		if err != nil {
			return result, err
		}
		result.Port = 443
		result.TLS = true
		result.Host = parseHost
		result.IP = iputil.GetDomainIP(parseHost)
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = getFingerprint(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), result.Headers)
		return result, nil

	default:
		result, err = retryhttpclient.GetHttpRequest(HTTPS_PREFIX + host)
		if err == nil {
			result.Port = 443
			strPort := ""
			if intPort, err := strconv.Atoi(parsePort); err == nil {
				result.Port = intPort
				strPort = ":" + parsePort
			}
			result.Host = parseHost
			result.IP = iputil.GetDomainIP(parseHost)
			result.TLS = true
			result.FullUrl = HTTPS_PREFIX + parseHost + strPort
			result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
			result.FingerPrint = getFingerprint(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), result.Headers)
			return result, err
		}

		result, err = retryhttpclient.GetHttpRequest(HTTP_PREFIX + host)
		if err == nil {
			if strings.Contains(result.Body, "<title>400 The plain HTTP request was sent to HTTPS port</title>") {
				result.Port = 443
				strPort := ""
				if intPort, err := strconv.Atoi(parsePort); err == nil {
					result.Port = intPort
					strPort = ":" + parsePort
				}
				result.Host = parseHost
				result.IP = iputil.GetDomainIP(parseHost)
				result.TLS = true
				result.FullUrl = HTTPS_PREFIX + parseHost + strPort
				result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
				result.FingerPrint = getFingerprint(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), result.Headers)
				return result, nil
			}
			result.Port = 80
			strPort := ""
			if intPort, err := strconv.Atoi(parsePort); err == nil {
				result.Port = intPort
				strPort = ":" + parsePort
			}
			result.Host = parseHost
			result.TLS = false
			result.IP = iputil.GetDomainIP(parseHost)
			result.FullUrl = HTTP_PREFIX + parseHost + strPort
			result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
			result.FingerPrint = getFingerprint(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), result.Headers)
			return result, nil
		}

	}

	return result, fmt.Errorf("scan host failed")
}

func getFingerprint(target string, body, raw, rawheader, faviconhash []byte, headers map[string]string) string {
	if nlo, err := libra.NewLibraOption(
		libra.SetTarget(target),
		libra.SetBody(body),
		libra.SetRaw(raw),
		libra.SetRawHeader(rawheader),
		libra.SetHeaders(headers),
		libra.SetFaviconhash(faviconhash),
	); err == nil && nlo != nil {
		res := nlo.Run()
		if res != nil && len(res.FingerResult) > 0 {
			return fingerprintSlice2String(res.FingerResult)
		}
	}
	return ""
}

func fingerprintSlice2String(f []string) string {
	fingerprint := ""
	if len(f) > 0 {
		for _, f := range f {
			fingerprint += "," + f
		}
		fingerprint = strings.TrimLeft(fingerprint, ",")
	}
	return fingerprint
}

func (r *Runner) Close() error {
	return os.RemoveAll(r.hostTempFile)
}
