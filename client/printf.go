package client

import (
	"fmt"
	"strings"
)

type prefix string

const (
	empty      = " "
	input      = ">"
	incoming   = "<"
	info       = "–"
	theEnd     = "\u2020"
	blockStart = "("
	blockEnd   = ")"
	raw        = "r"
)

func printF(p prefix, format string, c ...interface{}) {
	var (
		prefix, end string
	)

	prefix = strings.Repeat(" ", 2)
	end = fmt.Sprintf(" \n%s%s ", prefix, input)

	switch p {
	case blockStart, blockEnd:
		prefix = ""
		end = fmt.Sprintf(" \n")
	case input:
		prefix = strings.Repeat(" ", 2)
		end = ""
	case raw:
		fmt.Printf("\r%s\n", strings.Repeat(" ", 16))
		for _, l := range strings.Split(fmt.Sprintf(format, c...), "\n") {
			fmt.Printf("%s%s\n", strings.Repeat(" ", 4), l)
		}
		fmt.Print("\n")

		return
	}

	fmt.Printf("\r%s%s %s%s", prefix, p, fmt.Sprintf(format, c...), end)
}
