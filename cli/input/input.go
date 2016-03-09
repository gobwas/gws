package input

import (
	"bufio"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/common"
	"gopkg.in/readline.v1"
	"os"
)

type Message struct {
	Err  error
	Data []byte
}

func writeMessageNonBlocking(ch chan<- Message, m Message) {
	select {
	case ch <- m:
	default:
	}
}

func Readline(cfg *readline.Config) (r []byte, err error) {
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

func ReadFromStdReadline(done <-chan struct{}) (<-chan Message, error) {
	ch := make(chan Message)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:      cli.PaddingLeft + "> ",
		HistoryFile: "/tmp/gws_readline_client.tmp",
	})
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

func ReadFromStd(done <-chan struct{}) (<-chan Message, error) {
	ch := make(chan Message)
	reader := bufio.NewReader(os.Stdin)

	go func() {
		for {
			var msg Message
			b, err := reader.ReadBytes('\n')
			if err != nil {
				msg = Message{Err: err}
			} else {
				msg = Message{Data: b[:len(b)-1]}
			}

			select {
			case <-done:
			//
			default:
				ch <- msg
				if msg.Err != nil {
					return
				}
			}
		}
	}()

	return ch, nil
}
