package pyxis

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/cdncheck"
	"github.com/zan8in/godns"
	"github.com/zan8in/gologger"
	"github.com/zan8in/libra"
	"github.com/zan8in/pyxis/pkg/favicon"
	"github.com/zan8in/pyxis/pkg/http/retryhttpclient"
	"github.com/zan8in/pyxis/pkg/result"
	"github.com/zan8in/pyxis/pkg/util/iputil"
)

var defaultCdncheckTimeout = 3

type Runner struct {
	Options *Options

	ticker *time.Ticker
	wgscan sizedwaitgroup.SizedWaitGroup

	hostChan chan string

	ResultChan chan *result.HostResult
	Result     *result.Result

	hostTempFile string

	Phase Phase

	cdnchecker *cdncheck.CDNChecker

	// 新增：指纹识别专用并发控制
	fingerprintSemaphore chan struct{}
}

func NewRunner(options *Options) (*Runner, error) {
	var (
		err error
	)

	var cdnchecker *cdncheck.CDNChecker

	// 构建基础配置选项
	opts := []cdncheck.Option{
		cdncheck.WithRetries(options.Retries),
		cdncheck.WithTimeout(time.Duration(options.Timeout) * time.Second),
	}

	// 处理代理配置
	if options.Proxy != "" {
		proxyURL, err := url.Parse(options.Proxy)
		if err != nil {
			return nil, fmt.Errorf("解析代理URL失败: %v", err)
		}

		// 提取认证信息
		var auth *godns.ProxyAuth
		if proxyURL.User != nil {
			password, _ := proxyURL.User.Password()
			auth = &godns.ProxyAuth{
				Username: proxyURL.User.Username(),
				Password: password,
			}
		}

		// 根据代理类型添加相应选项
		switch strings.ToLower(proxyURL.Scheme) {
		case "socks5":
			opts = append(opts, cdncheck.WithSOCKS5Proxy(proxyURL.Host, auth))
		case "http", "https":
			opts = append(opts, cdncheck.WithHTTPProxy(proxyURL.Host, auth))
		default:
			return nil, fmt.Errorf("不支持的代理类型: %s，支持的类型: http, https, socks5", proxyURL.Scheme)
		}
	}

	cdnchecker = cdncheck.New(opts...)

	// 检查cdnchecker是否创建成功
	if cdnchecker == nil {
		return nil, fmt.Errorf("创建CDN检查器失败")
	}

	runner := &Runner{
		Options:    options,
		hostChan:   make(chan string),
		ResultChan: make(chan *result.HostResult),
		Result:     result.NewResult(),
		cdnchecker: cdnchecker,

		// 指纹识别并发限制为主并发的1/4，避免CPU过载
		fingerprintSemaphore: make(chan struct{}, calculateFingerprintConcurrency(options.RateLimit)),
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

func NewApiRunner(options *Options) (*Runner, error) {
	var (
		err error
	)

	runner := &Runner{
		Options: options,
	}

	if err = retryhttpclient.Init(&retryhttpclient.Options{
		Retries: options.Retries,
		Timeout: options.Timeout,
		Proxy:   options.Proxy,
	}); err != nil {
		return runner, err
	}

	return runner, err
}

func (r *Runner) Run() error {
	defer r.Close()

	go func() {
		if err := r.PreprocessHost(); err != nil {
			gologger.Error().Msg(err.Error())
		}
	}()

	// 使用 WaitGroup 等待 Listener 完成
	listenerWg := sync.WaitGroup{}
	listenerWg.Add(1)
	go func() {
		defer listenerWg.Done()
		r.Listener()
	}()

	r.start()

	r.Delay()

	// 等待 Listener 完全处理完所有结果
	listenerWg.Wait()

	r.WriteOutput()

	return nil
}

func (r *Runner) ApiRun() error {
	defer r.Close()

	go func() {
		if err := r.PreprocessHost(); err != nil {
			gologger.Error().Msg(err.Error())
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
		r.print(result)
	}
	r.Phase.Set(Done)
}

func (r *Runner) start() {
	defer close(r.ResultChan)
	r.Phase.Set(Scan)

	for host := range r.hostChan {
		// 等待 ticker，控制请求速率
		<-r.ticker.C

		r.wgscan.Add()
		go func(host string) {
			defer r.wgscan.Done()
			if rst, err := r.ScanHost(host); err == nil {
				r.ResultChan <- &rst
			} else {
				r.ResultChan <- &result.HostResult{Host: host, Flag: 1}
			}
		}(host)
	}
	r.wgscan.Wait()
}

func (r *Runner) ScanHost(host string) (result.HostResult, error) {
	if len(strings.TrimSpace(host)) == 0 {
		return result.HostResult{}, fmt.Errorf("host %q is empty", host)
	}

	var (
		err       error
		result    result.HostResult
		parseHost string
		parsePort string
	)

	// 如果启用了CDN选项，只进行CDN检测
	if r.Options.Cdn {
		// 解析主机名
		if strings.HasPrefix(host, HTTP_PREFIX) || strings.HasPrefix(host, HTTPS_PREFIX) {
			u, err := url.Parse(host)
			if err != nil {
				return result, err
			}
			parseHost = u.Hostname()
			// 设置 FullUrl 为原始输入
			result.FullUrl = host
		} else {
			// 移除端口号
			if strings.Contains(host, ":") {
				parts := strings.Split(host, ":")
				parseHost = parts[0]
			} else {
				parseHost = host
			}
			// 设置 FullUrl 为主机名（CDN模式下用作唯一标识）
			result.FullUrl = parseHost
		}

		// 只进行CDN检测
		result.Host = parseHost
		result.IP, result.Cdn, err = r.GetDomainIPWithCDN(parseHost)
		if err != nil {
			result.Flag = 1 // 标记为失败
			return result, err
		}
		result.Flag = 0 // 标记为成功
		return result, nil
	}

	if strings.HasPrefix(host, HTTPS_PREFIX) {
		result, err = retryhttpclient.Get(host)
		if err != nil {
			return result, err
		}
		result.Port = 443
		result.TLS = true
		result.Host = ""
		u, err := url.Parse(host)
		if err == nil {
			result.Host = u.Hostname()
			if ip, cdn, err := r.GetDomainIPWithCDN(u.Hostname()); err == nil {
				result.IP = ip
				result.Cdn = cdn
			} else {
				gologger.Warning().Msgf("Failed to get CDN info for %s: %v", u.Hostname(), err)
			}
		}
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = r.getFingerprintAsync(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), int32(result.StatusCode), result.Headers)
		return result, nil
	}

	if strings.HasPrefix(host, HTTP_PREFIX) {
		result, err = retryhttpclient.Get(host)
		if err != nil {
			return result, err
		}
		result.Port = 80
		result.TLS = false
		result.Host = ""
		u, err := url.Parse(host)
		if err == nil {
			result.Host = u.Hostname()
			if ip, cdn, err := r.GetDomainIPWithCDN(u.Hostname()); err == nil {
				result.IP = ip
				result.Cdn = cdn
			} else {
				gologger.Warning().Msgf("Failed to get CDN info for %s: %v", u.Hostname(), err)
			}
		}
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = r.getFingerprintAsync(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), int32(result.StatusCode), result.Headers)
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
		result, err = retryhttpclient.Get(HTTP_PREFIX + host)
		if err != nil {
			return result, err
		}
		result.Port = 80
		result.TLS = false
		result.Host = parseHost
		result.IP = iputil.GetDomainIP(parseHost)
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = r.getFingerprintAsync(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), int32(result.StatusCode), result.Headers)
		return result, nil

	case parsePort == "443":
		result, err = retryhttpclient.Get(HTTPS_PREFIX + host)
		if err != nil {
			return result, err
		}
		result.Port = 443
		result.TLS = true
		result.Host = parseHost
		if ip, cdn, err := r.GetDomainIPWithCDN(u.Hostname()); err == nil {
			result.IP = ip
			result.Cdn = cdn
		} else {
			gologger.Warning().Msgf("Failed to get CDN info for %s: %v", u.Hostname(), err)
		}
		result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
		result.FingerPrint = r.getFingerprintAsync(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), int32(result.StatusCode), result.Headers)
		return result, nil

	default:
		result, err = retryhttpclient.Get(HTTPS_PREFIX + host)
		if err == nil {
			result.Port = 443
			strPort := ""
			if intPort, err := strconv.Atoi(parsePort); err == nil {
				result.Port = intPort
				strPort = ":" + parsePort
			}
			result.Host = parseHost
			if ip, cdn, err := r.GetDomainIPWithCDN(u.Hostname()); err == nil {
				result.IP = ip
				result.Cdn = cdn
			} else {
				gologger.Warning().Msgf("Failed to get CDN info for %s: %v", u.Hostname(), err)
			}
			result.TLS = true
			result.FullUrl = HTTPS_PREFIX + parseHost + strPort
			result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
			result.FingerPrint = r.getFingerprintAsync(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), int32(result.StatusCode), result.Headers)
			return result, err
		}

		result, err = retryhttpclient.Get(HTTP_PREFIX + host)
		if err == nil {
			if strings.Contains(result.Body, "<title>400 The plain HTTP request was sent to HTTPS port</title>") {
				result.Port = 443
				strPort := ""
				if intPort, err := strconv.Atoi(parsePort); err == nil {
					result.Port = intPort
					strPort = ":" + parsePort
				}
				result.Host = parseHost
				if ip, cdn, err := r.GetDomainIPWithCDN(u.Hostname()); err == nil {
					result.IP = ip
					result.Cdn = cdn
				} else {
					gologger.Warning().Msgf("Failed to get CDN info for %s: %v", u.Hostname(), err)
				}
				result.TLS = true
				result.FullUrl = HTTPS_PREFIX + parseHost + strPort
				result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
				result.FingerPrint = r.getFingerprintAsync(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), int32(result.StatusCode), result.Headers)
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
			if ip, cdn, err := r.GetDomainIPWithCDN(u.Hostname()); err == nil {
				result.IP = ip
				result.Cdn = cdn
			} else {
				gologger.Warning().Msgf("Failed to get CDN info for %s: %v", u.Hostname(), err)
			}
			result.FullUrl = HTTP_PREFIX + parseHost + strPort
			result.FaviconHash = favicon.FaviconHash(result.FullUrl, result.Body)
			result.FingerPrint = r.getFingerprintAsync(result.FullUrl, result.RawBody, result.Raw, result.RawHeader, []byte(result.FaviconHash), int32(result.StatusCode), result.Headers)
			return result, nil
		}

	}

	return result, fmt.Errorf("scan host failed")
}

func getFingerprint(target string, body, raw, rawheader, faviconhash []byte, status int32, headers map[string]string) string {
	if nlo, err := libra.NewLibraOption(
		libra.SetStatus(status),
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
	if r.ticker != nil {
		r.ticker.Stop()
	}
	return os.RemoveAll(r.hostTempFile)
}

// GetDomainIPWithCDN 获取域名的IP和CDN
// @param domain 域名
// @return
//
//	ip 域名的IP（多个IP用逗号分隔）
//	cdn 域名的CDN信息
//	err 错误
func (r *Runner) GetDomainIPWithCDN(domain string) (string, string, error) {
	if domain == "" {
		return "", "", fmt.Errorf("域名不能为空")
	}

	// 统一的CDN信息格式化函数
	formatCDNInfo := func(isCDN bool, provider string) string {
		if !isCDN {
			return ""
		}
		if provider != "" && !strings.Contains(provider, ",") {
			return "CDN:" + provider
		}
		return "CDN"
	}

	// 处理IP输入
	if iputil.IsIP(domain) {
		result, _ := r.cdnchecker.CheckIP(domain)
		return domain, formatCDNInfo(result.IsCDN, result.Provider), nil
	}

	// 处理域名输入
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultCdncheckTimeout)*time.Second)
	defer cancel()

	result, err := r.cdnchecker.CheckDomain(ctx, domain)
	if err != nil {
		return "", "", err
	}

	// 特殊情况处理
	// 如果不是CDN但是有两个IP，认为是负载均衡
	if !result.IsCDN && len(result.IPs) == 2 {
		result.IsCDN = true
		result.Provider = "负载均衡"
	}

	return strings.Join(result.IPs, ","), formatCDNInfo(result.IsCDN, result.Provider), nil
}

// 新增：异步指纹识别函数
func (r *Runner) getFingerprintAsync(target string, body, raw, rawheader, faviconhash []byte, status int32, headers map[string]string) string {
	resultChan := make(chan string, 1)

	go func() {
		// 获取信号量，限制并发数
		r.fingerprintSemaphore <- struct{}{}
		defer func() { <-r.fingerprintSemaphore }()

		// 执行指纹识别
		fingerprint := getFingerprint(target, body, raw, rawheader, faviconhash, status, headers)
		resultChan <- fingerprint
	}()

	// 设置超时，避免长时间阻塞
	select {
	case fingerprint := <-resultChan:
		return fingerprint
	case <-time.After(5 * time.Second):
		return "" // 超时返回空字符串
	}
}

func calculateFingerprintConcurrency(rateLimit int) int {
	cpuCores := runtime.NumCPU()

	// 根据CPU核心数动态调整除数
	var divisor int
	switch {
	case cpuCores <= 2:
		divisor = 80 // 极保守
	case cpuCores <= 4:
		divisor = 60 // 保守
	case cpuCores <= 8:
		divisor = 50 // 中等
	case cpuCores <= 12:
		divisor = 40 // 你当前的设置
	case cpuCores <= 16:
		divisor = 35 // 稍微激进
	default:
		divisor = 30 // 高性能CPU
	}

	return max(1, rateLimit/divisor)
}
