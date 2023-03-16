package pyxis

import (
	"fmt"

	"github.com/zan8in/gologger"
)

var Version = "0.1.1"

var banner = fmt.Sprintf(`
┌─┐┬ ┬─┐ ┬┬┌─┐
├─┘└┬┘┌┴┬┘│└─┐
┴   ┴ ┴ └─┴└─┘ %s
`, Version)

func ShowBanner() {
	gologger.Print().Msgf("%s\n", banner)
	gologger.Print().Msgf("\thttps://github.com/zan8in/pyxis\n\n")
}
