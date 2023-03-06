package pyxis

import (
	"github.com/zan8in/pyxis/pkg/result"
)

type Scanner struct {
	options *Options
	Result  *result.Result
}

func NewScanner(options *Options) (*Scanner, error) {

	if options.Host == nil && options.HostsFile == "" {
		return nil, errNoInputList
	}

	if options.Timeout == 0 {
		options.Timeout = DefaultTimeout
	}

	if options.RateLimit <= 0 {
		options.autoChangeRateLimit()
	}

	if options.Retries <= 0 {
		options.Retries = DefaultRetries
	}

	scanner := &Scanner{
		options: options,
		Result:  result.NewResult(),
	}

	return scanner, nil
}

func (s *Scanner) Run() error {
	runner, err := NewRunner(s.options)
	if err != nil {
		return err
	}

	runner.ApiRun()

	if runner.Result.HasHostResult() {
		for hostResult := range runner.Result.GetHostResult() {
			s.Result.AddHostResult(hostResult)
		}
	}

	return nil
}
