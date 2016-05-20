package cli

import (
	"fmt"
	"strings"
)

type prefix string

const (
	PrefixEmpty      = " "
	PrefixInput      = ">"
	PrefixIncoming   = "<"
	PrefixInfo       = "–"
	PrefixTheEnd     = "\u2020"
	PrefixBlockStart = "("
	PrefixBlockEnd   = ")"
	PrefixRaw        = "r"
)

const (
	PaddingLeft = "  "
)

func Printf(prefix prefix, format string, c ...interface{}) {
	var (
		padLeft, end string
	)

	padLeft = PaddingLeft
	end = fmt.Sprintf(" \n%s%s ", padLeft, PrefixInput)

	switch prefix {
	case PrefixBlockStart, PrefixBlockEnd:
		padLeft = ""
		end = fmt.Sprintf(" \n")
	case PrefixInput:
		end = ""
	case PrefixRaw:
		fmt.Printf("\r%s\n", strings.Repeat(" ", 16))
		for _, l := range strings.Split(fmt.Sprintf(format, c...), "\n") {
			fmt.Printf("%s%s\n", strings.Repeat(" ", 4), l)
		}
		fmt.Print("\n")

		return
	}

	fmt.Printf("\r%s%s %s%s", padLeft, prefix, fmt.Sprintf(format, c...), end)
}
