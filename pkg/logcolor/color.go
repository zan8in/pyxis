package logcolor

import (
	"github.com/gookit/color"
)

type Color struct {
	Title       func(a ...any) string
	Fingerprint func(a ...any) string
	Faviconhash func(a ...any) string
	IP          func(a ...any) string
}

var LogColor *Color

func init() {
	if LogColor == nil {
		LogColor = NewColor()
	}
}

func NewColor() *Color {
	return &Color{
		Title:       color.Green.Render,
		Fingerprint: color.Comment.Render,
		Faviconhash: color.Yellow.Render,
		IP:          color.Cyan.Render,
	}
}
