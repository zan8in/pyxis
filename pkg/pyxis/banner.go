package pyxis

import (
	"github.com/zan8in/gologger"
)

var Version = "0.1.2"

func ShowBanner() {
	gologger.Print().Msgf("\n|||\tP Y X I S\t|||\t%s\n\n", Version)
}
