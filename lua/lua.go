package lua

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/gws/bufio"
	"github.com/gobwas/gws/cli"
	"github.com/gobwas/gws/cli/color"
	"github.com/gobwas/gws/config"
	"github.com/gobwas/gws/display"
	"github.com/gobwas/gws/ev"
	evWS "github.com/gobwas/gws/ev/ws"
	modRuntime "github.com/gobwas/gws/lua/mod/runtime"
	modStat "github.com/gobwas/gws/lua/mod/stat"
	modTime "github.com/gobwas/gws/lua/mod/time"
	modWS "github.com/gobwas/gws/lua/mod/ws"
	"github.com/gobwas/gws/lua/script"
	"github.com/gobwas/gws/lua/util"
	"github.com/gobwas/gws/stat"
)

var scriptFile = flag.String("path", "", "path to lua script")

func initRunTime(loop *ev.Loop, c config.Config) *modRuntime.Runtime {
	rtime := modRuntime.New(loop)
	rtime.Set("url", c.URI)
	rtime.Set("listen", c.Addr)
	rtime.Set("headers", util.HeadersToMap(c.Headers))

	environment := os.Environ()
	env := make(map[string]string, len(environment))
	for _, e := range environment {
		pair := strings.Split(e, "=")
		env[pair[0]] = pair[1]
	}
	rtime.Set("env", env)

	return rtime
}

func Go(c config.Config) error {
	var code string
	if script, err := ioutil.ReadFile(*scriptFile); err != nil {
		return err
	} else {
		code = string(script)
	}

	stats := stat.New()

	luaOutputBuffer := bytes.NewBuffer(make([]byte, 0, 1<<13))
	luaStdout := bufio.NewWriter(luaOutputBuffer, 1<<13)

	systemStdout := bytes.NewBuffer(make([]byte, 0, 1024))

	printer := display.NewDisplay(os.Stderr, display.Config{
		TabSize:  4,
		Interval: time.Millisecond * 100,
	})
	printer.Row().Col(-1, -1, func() string {
		return stats.Pretty()
	})
	printer.Row().Col(256, 10, func() (str string) {
		luaStdout.Dump()
		str = luaOutputBuffer.String()
		luaOutputBuffer.Reset()
		return
	})
	printer.Row().Col(256, 3, func() (str string) {
		str = systemStdout.String()
		return
	})
	printer.On()
	defer printer.Off()
	defer printer.Render()

	cancel := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		s := <-c
		fmt.Fprintln(systemStdout, color.Cyan(cli.PrefixTheEnd), s.String())
		fmt.Fprintln(systemStdout, color.Cyan("stopping softly.."))
		close(cancel)
		s = <-c
		fmt.Fprintln(systemStdout, color.Red(cli.PrefixTheEnd), color.Yellow(s.String()+"x2"))
		fmt.Fprintln(systemStdout, color.Red("stopping hardly.."))
		printer.Off()
		os.Exit(1)
	}()

	luaScript := script.New()
	defer luaScript.Shutdown()

	luaScript.HijackOutput(bufio.NewPrefixWriter(luaStdout, color.Green("master > ")))

	loop := ev.NewLoop()

	loopServerHandler := evWS.NewServerHandler()
	loop.Register(evWS.NewClientHandler(), 100)
	loop.Register(loopServerHandler, 101)

	sharedStat := modStat.New(stats)

	var wg sync.WaitGroup
	var threads int
	rtime := initRunTime(loop, c)
	rtime.SetForkFn(func() error {
		go func(id int) {
			defer wg.Done()

			luaScript := script.New()
			defer luaScript.Shutdown()
			luaScript.HijackOutput(bufio.NewPrefixWriter(luaStdout, color.Green(fmt.Sprintf("thread %.2d > ", id))))

			loop := ev.NewLoop()
			loop.Register(evWS.NewClientHandler(), 100)
			loop.Register(loopServerHandler, 101)

			rtime := initRunTime(loop, c)
			rtime.Set("id", id)

			luaScript.Preload("runtime", rtime)
			luaScript.Preload("stat", sharedStat)
			luaScript.Preload("time", modTime.New(loop))
			luaScript.Preload("ws", modWS.New(loop))

			err := luaScript.Do(code)
			if err != nil {
				log.Printf("run forked lua script error: %s", err)
			}

			loop.Run()
			loop.Teardown(func() {
				rtime.Emit("exit")
			})

			waitLoop(cancel, loop)
		}(threads)

		wg.Add(1)
		threads++

		return nil
	})

	luaScript.Preload("runtime", rtime)
	luaScript.Preload("stat", sharedStat)
	luaScript.Preload("time", modTime.New(loop))
	luaScript.Preload("ws", modWS.New(loop))

	err := luaScript.Do(code)
	if err != nil {
		log.Printf("run lua script error: %s", err)
		return err
	}

	loop.Run()
	loop.Teardown(func() {
		wg.Wait()
		rtime.Emit("exit")
	})

	waitLoop(cancel, loop)
	wg.Wait()

	return nil
}

func waitLoop(cancel chan struct{}, loop *ev.Loop) {
	select {
	case <-loop.Done():
	case <-cancel:
		loop.Stop()
		shutdown := time.NewTimer(time.Second * 4)
		select {
		case <-loop.Done():
		case <-shutdown.C:
			loop.Shutdown()
		}
	}
}
