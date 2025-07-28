package cdnlib

import (
	"fmt"
	"net"
	"sync"
)

// CDNProvider 定义了已知的CDN提供商的CIDR或ASN信息
var CDNProvider = map[string][]string{
	"Cloudflare":        {"103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22", "104.16.0.0/12", "108.162.192.0/18", "131.0.72.0/22", "141.101.64.0/18", "162.158.0.0/15", "172.64.0.0/13", "173.245.48.0/20", "188.114.96.0/20", "190.93.240.0/20", "197.234.240.0/22", "198.41.128.0/17"},
	"Akamai":            {"23.32.0.0/11", "104.64.0.0/10", "184.24.0.0/13", "184.50.0.0/15", "184.84.0.0/14", "2.16.0.0/13", "95.100.0.0/15", "23.0.0.0/12", "96.16.0.0/15", "72.246.0.0/15"},
	"Amazon CloudFront": {"54.182.0.0/16", "54.192.0.0/16", "54.230.0.0/16", "54.239.128.0/18", "54.239.192.0/19", "99.84.0.0/16", "205.251.192.0/19", "52.124.128.0/17", "204.246.164.0/22", "204.246.168.0/22", "204.246.174.0/23", "204.246.176.0/20", "13.32.0.0/15", "13.224.0.0/14", "13.35.0.0/16", "204.246.172.0/24", "204.246.173.0/24"},
	"Fastly":            {"23.235.32.0/20", "43.249.72.0/22", "103.244.50.0/24", "103.245.222.0/23", "103.245.224.0/24", "104.156.80.0/20", "146.75.0.0/16", "151.101.0.0/16", "157.52.64.0/18", "167.82.0.0/17", "167.82.128.0/20", "167.82.160.0/20", "167.82.224.0/20", "172.111.64.0/18", "185.31.16.0/22", "199.27.72.0/21", "199.232.0.0/16"},
	"Google":            {"34.64.0.0/10", "34.128.0.0/10", "35.184.0.0/13", "35.192.0.0/14", "35.196.0.0/15", "35.198.0.0/16", "35.199.0.0/17", "35.199.128.0/18", "35.200.0.0/13", "35.208.0.0/12", "35.224.0.0/12", "35.240.0.0/13", "64.233.160.0/19", "66.102.0.0/20", "66.249.64.0/19", "70.32.128.0/19", "72.14.192.0/18", "74.125.0.0/16", "108.177.0.0/17", "142.250.0.0/15", "172.217.0.0/16", "173.194.0.0/16", "209.85.128.0/17", "216.58.192.0/19", "216.239.32.0/19"},
	"Microsoft Azure":   {"13.64.0.0/11", "13.96.0.0/13", "13.104.0.0/14", "20.33.0.0/16", "20.34.0.0/15", "20.36.0.0/14", "20.40.0.0/13", "20.48.0.0/12", "20.64.0.0/10", "20.128.0.0/16", "20.135.0.0/16", "20.136.0.0/16", "20.143.0.0/16", "20.144.0.0/14", "20.150.0.0/15", "20.152.0.0/16", "20.153.0.0/16", "20.157.0.0/16", "20.158.0.0/15", "20.160.0.0/12", "20.176.0.0/14", "20.180.0.0/14", "20.184.0.0/13", "20.192.0.0/10"},
}

// DomainCheckResult 表示域名检查的结果
type DomainCheckResult struct {
	Domain   string
	IP       string
	IsCDN    bool
	Provider string
	Reason   string
}

// IsCDNIP 检查IP是否属于CDN提供商
func IsCDNIP(ip string) (bool, string) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false, ""
	}

	for provider, cidrs := range CDNProvider {
		for _, cidr := range cidrs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if ipNet.Contains(ipAddr) {
				return true, provider
			}
		}
	}

	return false, ""
}

// CheckDomain 检查域名是否使用CDN，返回检查结果
func CheckDomain(domain string) ([]DomainCheckResult, error) {
	var results []DomainCheckResult

	ips, err := net.LookupIP(domain)
	if err != nil {
		return []DomainCheckResult{{Domain: domain, IsCDN: false, Reason: "解析失败"}}, err
	}

	// 检查是否有多个IP（可能是CDN的一个特征）
	multipleIPs := len(ips) > 1

	// 检查IP是否在已知的CDN IP范围内
	for _, ip := range ips {
		ipStr := ip.String()
		isCDN, provider := IsCDNIP(ipStr)

		result := DomainCheckResult{
			Domain: domain,
			IP:     ipStr,
			IsCDN:  isCDN,
		}

		if isCDN {
			result.Provider = provider
			result.Reason = "已知CDN提供商"
		} else if multipleIPs {
			// 如果有多个IP但不在已知CDN列表中，可能仍是CDN
			result.IsCDN = true
			result.Reason = "多IP"
		}

		results = append(results, result)
	}

	return results, nil
}

// CheckDomainsConcurrent 并发检查多个域名
func CheckDomainsConcurrent(domains []string) []DomainCheckResult {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var allResults []DomainCheckResult

	for _, domain := range domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			results, err := CheckDomain(d)
			if err == nil {
				mutex.Lock()
				allResults = append(allResults, results...)
				mutex.Unlock()
			} else {
				mutex.Lock()
				allResults = append(allResults, DomainCheckResult{Domain: d, IsCDN: false, Reason: "解析失败"})
				mutex.Unlock()
			}
		}(domain)
	}

	wg.Wait()
	return allResults
}

// PrintResults 打印检查结果
func PrintResults(results []DomainCheckResult) {
	fmt.Println("域名\tIP\t是否CDN")
	fmt.Println("----------------------------------------")

	for _, result := range results {
		cdnStatus := "否"
		if result.IsCDN {
			if result.Provider != "" {
				cdnStatus = fmt.Sprintf("是 (%s)", result.Provider)
			} else {
				cdnStatus = fmt.Sprintf("可能是 (%s)", result.Reason)
			}
		}
		fmt.Printf("%s\t%s\t%s\n", result.Domain, result.IP, cdnStatus)
	}
}
