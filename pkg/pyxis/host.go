package pyxis

import (
	"fmt"
	"io"
	"os"
)

func (r *Runner) PreprocessHost() error {
	var err error

	hostTemp, err := os.CreateTemp("", HostTempFile)
	if err != nil {
		return err
	}
	defer hostTemp.Close()

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

	r.hostTempFile = hostTemp.Name()

	return err
}
