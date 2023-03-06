package pyxis

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/gologger"
	"github.com/zan8in/pyxis/pkg/http/retryhttpclient"
	"github.com/zan8in/pyxis/pkg/result"
)

type Runner struct {
	Options *Options

	ticker *time.Ticker
	wgscan sizedwaitgroup.SizedWaitGroup

	hostChan chan string

	hostTempFile string
}

func NewRunner(options *Options) (*Runner, error) {
	var (
		err error
	)

	runner := &Runner{
		Options:  options,
		hostChan: make(chan string),
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

	r.start()

	return nil
}

func (r *Runner) start() {
	for host := range r.hostChan {
		r.wgscan.Add()
		go func(host string) {
			if result, err := r.scanHost(host); err == nil {
				print(result)
			}
		}(host)
	}
	r.wgscan.Wait()
}

func print(result result.HostResult) {
	fmt.Println(
		result.FullUrl,
		result.Title,
		result.TLS,
		result.Host,
		result.Port,
		result.IP,
		result.StatusCode,
		result.ResponseTime,
		result.ContentLength,
	)
}

func (r *Runner) scanHost(host string) (result.HostResult, error) {
	defer r.wgscan.Done()

	// fmt.Println("\n", host, "---------------\n")

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
			result.Host = u.Host
		}
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
			result.Host = u.Host
		}
		return result, nil
	}

	u, err := url.Parse(HTTP_PREFIX + host)
	if err != nil {
		return result, err
	}
	parseHost = u.Host
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
		return result, nil

	case parsePort == "443":
		result, err = retryhttpclient.GetHttpRequest(HTTPS_PREFIX + host)
		if err == nil {
			return result, err
		}
		result.Port = 443
		result.TLS = true
		result.Host = parseHost
		return result, nil

	default:
		result, err = retryhttpclient.GetHttpRequest(HTTPS_PREFIX + host)
		if err == nil {
			result.Port = 0
			strPort := ""
			if intPort, err := strconv.Atoi(parsePort); err == nil {
				result.Port = intPort
				strPort = ":" + parsePort
			}
			result.Host = parseHost
			result.TLS = true
			result.FullUrl = HTTPS_PREFIX + parseHost + strPort
			return result, err
		}

		result, err = retryhttpclient.GetHttpRequest(HTTP_PREFIX + host)
		if err == nil {
			if strings.Contains(result.Body, "<title>400 The plain HTTP request was sent to HTTPS port</title>") {
				result.Port = 0
				strPort := ""
				if intPort, err := strconv.Atoi(parsePort); err == nil {
					result.Port = intPort
					strPort = ":" + parsePort
				}
				result.TLS = true
				result.FullUrl = HTTPS_PREFIX + parseHost + strPort
				return result, nil
			}
			result.Port = 0
			strPort := ""
			if intPort, err := strconv.Atoi(parsePort); err == nil {
				result.Port = intPort
				strPort = ":" + parsePort
			}
			result.TLS = false
			result.FullUrl = HTTP_PREFIX + parseHost + strPort
			return result, nil
		}

	}

	return result, fmt.Errorf("scan host failed")
}

func (r *Runner) unKonwHttpProtocol(target string) bool {
	if len(strings.TrimSpace(target)) == 0 {
		return false
	}
	if strings.HasPrefix(target, HTTPS_PREFIX) {
		return false
	}
	if strings.HasPrefix(target, HTTP_PREFIX) {
		return false
	}
	u, err := url.Parse(HTTP_PREFIX + target)
	if err != nil {
		return true
	}
	parsePort := u.Port()

	if parsePort == "80" {
		return false
	}

	if parsePort == "443" {
		return false
	}

	return true
}

func (r *Runner) Close() error {
	return os.RemoveAll(r.hostTempFile)
}
