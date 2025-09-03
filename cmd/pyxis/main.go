package main

import (
	"os"
	"os/signal"
	"syscall"
	"github.com/zan8in/gologger"
	"github.com/zan8in/pyxis/pkg/pyxis"
)

func main() {
	// 设置信号处理
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		gologger.Info().Msg("收到退出信号，正在停止...")
		os.Exit(1)
	}()

	options := pyxis.ParseOptions()

	runner, err := pyxis.NewRunner(options)
	if err != nil {
		gologger.Fatal().Msg(err.Error())
	}

	runner.Run()
}
