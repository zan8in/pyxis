package result

import (
	"sync"
)

const (
	// Success is returned when the operation was successful.
	Success = iota
	// Error is returned when the operation failed.
	Failed
)

type Result struct {
	sync.RWMutex
	hosts map[string]*HostResult
}

type HostResult struct {
	Flag          int    //
	FullUrl       string // The full URL
	Host          string // example.com or ip addr
	Port          int    // port number
	TLS           bool   // true if TLS
	IP            string // IP address
	Title         string // title of the response
	Body          string // body of the response
	StatusCode    int    // status code of the response
	ContentLength int64  // content length of the response
	ResponseTime  int64  // time of the response
	FaviconHash   string // favicon hash
	FingerPrint   string
	RawBody       []byte //
	Raw           []byte // raw
	RawHeader     []byte // header
	Headers       map[string]string
}

func NewResult() *Result {
	return &Result{
		hosts: make(map[string]*HostResult),
	}
}

func (r *Result) GetHostResult() chan *HostResult {
	r.Lock()

	out := make(chan *HostResult)

	go func() {
		defer close(out)
		defer r.Unlock()

		for _, hostResult := range r.hosts {
			out <- hostResult
		}
	}()

	return out
}

func (r *Result) HasHostResult() bool {
	r.RLock()
	defer r.RUnlock()

	return len(r.hosts) > 0
}

func (r *Result) AddHostResult(hostResult *HostResult) {
	r.Lock()
	defer r.Unlock()

	r.hosts[hostResult.Host] = hostResult
}

func (r *Result) AddHostResultSlice(hostResult []*HostResult) {
	r.Lock()
	defer r.Unlock()

	for _, hostResult := range hostResult {
		r.hosts[hostResult.Host] = hostResult
	}
}

func (r *Result) SetHostResult(host string, hostResult *HostResult) {
	r.Lock()
	defer r.Unlock()

	r.hosts[host] = hostResult
}
