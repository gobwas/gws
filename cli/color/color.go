package color

import (
	"github.com/fatih/color"
)

var (
	Red     = color.New(color.FgRed).SprintFunc()
	Magenta = color.New(color.FgMagenta).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
)
