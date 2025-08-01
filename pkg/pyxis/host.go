package pyxis

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/remeh/sizedwaitgroup"
	"github.com/zan8in/gologger"
	"github.com/zan8in/pyxis/pkg/util/iputil"
)

func (r *Runner) PreprocessHost() error {
	var (
		err error
	)

	hostTemp, err := os.CreateTemp("", HostTempFile)
	if err != nil {
		return err
	}
	// 确保文件被关闭和删除
	fileName := hostTemp.Name()
	defer func() {
		hostTemp.Close()
		// 如果函数正常结束，不删除文件，因为后续还需要使用
		// 文件将在Runner.Close()中被删除
	}()

	if len(r.Options.Host) > 0 {
		for _, host := range r.Options.Host {
			if _, err := fmt.Fprintf(hostTemp, "%s\n", host); err != nil {
				continue
			}
		}
	}

	if len(r.Options.HostsFile) > 0 {
		f, err := os.Open(r.Options.HostsFile)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(hostTemp, f); err != nil {
			return err
		}
	}

	r.hostTempFile = fileName

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	defer close(r.hostChan)

	wg := sizedwaitgroup.New(r.Options.RateLimit)
	s := bufio.NewScanner(f)
	for s.Scan() {
		wg.Add()
		go func(target string) {
			defer wg.Done()
			if err := r.processTarget(target); err != nil {
				gologger.Warning().Msgf("%s\n", err)
			}
		}(s.Text())
	}
	wg.Wait()

	return err
}

func (r *Runner) processTarget(target string) error {
	var err error

	target = strings.TrimSpace(target)
	if len(target) == 0 {
		return nil
	}

	if iputil.IsCIDR(target) {
		return nil
	}

	r.hostChan <- target
	// gologger.Info().Msg(target)
	return err
}
