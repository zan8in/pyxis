package main

import (
	"github.com/zan8in/gologger"
	"github.com/zan8in/pyxis/pkg/pyxis"
)

func main() {
	options := pyxis.ParseOptions()

	runner, err := pyxis.NewRunner(options)
	if err != nil {
		gologger.Fatal().Msg(err.Error())
	}

	runner.Run()
}
