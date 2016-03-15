package server

import (
	"github.com/chzyer/readline"
	"github.com/gobwas/gws/cli/color"
	"github.com/gobwas/gws/cli/input"
	"github.com/gobwas/gws/ws"
)

func DevNullResponder(t ws.Kind, msg []byte) ([]byte, error) {
	return nil, nil
}

func EchoResponder(t ws.Kind, msg []byte) ([]byte, error) {
	return msg, nil
}

func MirrorResponder(t ws.Kind, msg []byte) (r []byte, err error) {
	if t != ws.TextMessage {
		return
	}

	resp := []rune(string(msg))
	for i, l := 0, len(resp)-1; i < len(resp)/2; i, l = i+1, l-1 {
		resp[i], resp[l] = resp[l], resp[i]
	}

	return []byte(string(resp)), nil
}

func PromptResponder(t ws.Kind, msg []byte) (r []byte, err error) {
	r, err = input.Readline(&readline.Config{
		Prompt:      color.Green("> "),
		HistoryFile: "/tmp/gws_readline_server.tmp",
	})
	if err == readline.ErrInterrupt {
		return nil, nil
	}

	return

	//	fmt.Printf("\033[F")
}
