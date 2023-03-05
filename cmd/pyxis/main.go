package main

import (
	"github.com/zan8in/gologger"
	"github.com/zan8in/pyxis/pkg/http/retryhttpclient"
)

func main() {
	gologger.Info().Msg("Hello Pyxis")

	retryhttpclient.Init(&retryhttpclient.Options{
		Retries: 3,
		Timeout: 10,
		Proxy:   "",
	})

	resp, headers, flag, err := retryhttpclient.FingerPrintGet("http://example.com")
	gologger.Info().Msgf("%s %v %d %v", resp, headers, flag, err)
}
