package input

import (
	"bufio"
	"github.com/chzyer/readline"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/common"
	"os"
)

type Message struct {
	Err  error
	Data []byte
}

func ReadLine(cfg *readline.Config) (r []byte, err error) {
	rl, err := readline.NewEx(cfg)
	if err != nil {
		return
	}
	defer rl.Close()

	line, err := rl.Readline()
	if err != nil {
		return
	}

	return []byte(line), nil
}

func ReadLineAsync(done <-chan struct{}, cfg *readline.Config) (<-chan Message, error) {
	ch := make(chan Message)

	rl, err := readline.NewEx(cfg)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			var msg Message
			line, err := rl.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					msg = Message{Err: common.ErrExitZero}
				} else {
					msg = Message{Err: err}
				}

			} else {
				msg = Message{Data: []byte(line)}
			}

			select {
			case <-done:
			//
			default:
				ch <- msg
				if msg.Err != nil {
					rl.Close()
					return
				}
			}
		}
	}()

	return ch, nil
}
