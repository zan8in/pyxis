package pyxis

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
	"github.com/zan8in/goflags"
	"github.com/zan8in/gologger"
)

type Options struct {
	Host      goflags.StringSlice // Host is the single host or comma-separated list of hosts to find ports for
	HostsFile string              // HostsFile is the file containing list of hosts to find port for

	Retries   int    // Retries is the number of retries for the port
	RateLimit int    // RateLimit is the rate of port scan requests
	Timeout   int    // Timeout is the seconds to wait for ports to respond
	Proxy     string // http/socks5 proxy to use
	Output    string // Output is the file to write found ports to.

	Silent bool // Silent is the flag to show only results
	Cdn    bool
	Clear  bool // Clear is the flag to show only successful results
}

func ParseOptions() *Options {

	options := &Options{}

	flagSet := goflags.NewFlagSet()
	flagSet.SetDescription(`Pyxis`)

	flagSet.CreateGroup("input", "Input",
		flagSet.StringSliceVarP(&options.Host, "t", "target", nil, "hosts to scan ports for (comma-separated)", goflags.NormalizedStringSliceOptions),
		flagSet.StringVarP(&options.HostsFile, "T", "target-file", "", "list of hosts to scan ports (file)"),
	)

	flagSet.CreateGroup("output", "Output",
		flagSet.StringVarP(&options.Output, "output", "o", "", "file to write output to (optional), support format: txt,csv,json"),
	)

	flagSet.CreateGroup("optimization", "Optimization",
		flagSet.IntVar(&options.Retries, "retries", DefaultRetries, "number of retries for the port scan"),
		flagSet.IntVar(&options.Timeout, "timeout", DefaultTimeout, "millisecond to wait before timing out"),
		flagSet.BoolVar(&options.Cdn, "cdn", false, "check if the host is a cdn"),
		flagSet.BoolVar(&options.Silent, "silent", false, "only results only"),
		flagSet.BoolVar(&options.Clear, "clear", false, "only show successful results"),
	)

	flagSet.CreateGroup("rate-limit", "Rate-limit",
		flagSet.IntVar(&options.RateLimit, "rate", DefaultRateLimit, "packets to send per second"),
	)

	flagSet.CreateGroup("proxy", "Proxy",
		flagSet.StringVar(&options.Proxy, "proxy", "", "list of http/socks5 proxy to use (comma separated or file input)"),
	)

	_ = flagSet.Parse()

	if !options.Silent {
		ShowBanner()
	}

	err := options.validateOptions()
	if err != nil {
		gologger.Fatal().Msgf("Program exiting: %s\n", err)
	}

	return options
}

var (
	errNoInputList = errors.New("no input list provided")
	errZeroValue   = errors.New("cannot be zero")
)

func (options *Options) validateOptions() (err error) {

	if options.Host == nil && options.HostsFile == "" {
		return errNoInputList
	}

	if options.Timeout == 0 {
		return errors.Wrap(errZeroValue, "timeout")
	}

	if options.RateLimit <= 0 {
		return errors.Wrap(errZeroValue, "rate")
	} else if options.RateLimit == DefaultRateLimit {
		options.autoChangeRateLimit()
	}

	return err
}

func (options *Options) autoChangeRateLimit() {
	NumCPU := runtime.NumCPU()

	// 更加保守的设置，防止初始爆发
	switch {
	case NumCPU <= 2:
		options.RateLimit = 10 // 进一步降低
	case NumCPU <= 4:
		options.RateLimit = 20 // 进一步降低
	case NumCPU <= 8:
		options.RateLimit = 30 // 进一步降低
	case NumCPU <= 16:
		options.RateLimit = 40 // 进一步降低
	default:
		options.RateLimit = 50 // 进一步降低
	}

	fmt.Printf("[INFO] Auto-adjusted rate limit to %d based on %d CPU cores\n", options.RateLimit, NumCPU)
}
