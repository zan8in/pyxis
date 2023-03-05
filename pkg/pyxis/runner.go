package pyxis

import (
	"time"

	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/pyxis/pkg/http/retryhttpclient"
)

type Runner struct {
	Options *Options

	ticker *time.Ticker
	wgscan sizedwaitgroup.SizedWaitGroup

	hostTempFile string
}

func NewRunner(options *Options) (*Runner, error) {
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

	runner.wgscan = sizedwaitgroup.New(options.RateLimit)
	runner.ticker = time.NewTicker(time.Second / time.Duration(options.RateLimit))

	return runner, err
}
