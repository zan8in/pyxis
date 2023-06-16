package logcolor

import (
	"github.com/gookit/color"
)

type Color struct {
	Status        func(a ...any) string
	Title         func(a ...any) string
	Fingerprint   func(a ...any) string
	Faviconhash   func(a ...any) string
	IP            func(a ...any) string
	Failed        func(a ...any) string
	ContentLength func(a ...any) string
}

var LogColor *Color

func init() {
	if LogColor == nil {
		LogColor = NewColor()
	}
}

func NewColor() *Color {
	return &Color{

		Status:        color.Yellow.Render,
		Title:         color.Green.Render,
		Fingerprint:   color.Comment.Render,
		Faviconhash:   color.Magenta.Render,
		IP:            color.Cyan.Render,
		Failed:        color.Gray.Render,
		ContentLength: color.Gray.Render,
	}
}
