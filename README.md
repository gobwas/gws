# :sparkles: gws

> CLI tool for websocket testing

![Demo](https://cdn.rawgit.com/gobwas/gws/static/demo.gif)

## Install

```shell
go get github.com/gobwas/gws
```

## Usage exmaples

Connect to the websocket server:

```shell
gws client -url="ws://my.cool.address"
```

Run simple server and type response messages in terminal:

```shell
gws server -listen=":8888" -response=prompt
```

Or just simple echo:

```shell
gws server -listen=":8888" -response=echo
```

Run lua script:

```shell
gws script -path=./my_cool_script.lua
```

Usage info:

```shell
Usage of gws:
gws client|server|script [options]
options:
  -header string
        list of headers to be passed during handshake (both in client or server)
        format:
                { pair[ ";" pair...] },
        pair:
                { key ":" value }
  -listen string
        address to listen (default ":3000")
  -origin string
        use this glob pattern for server origin checks
  -path string
        path to lua script
  -response value
        how should server response on message (echo, mirror, prompt, null) (default null)
  -retry int
        try to reconnect x times (default 1)
  -statd duration
        server statistics dump interval (default 1s)
  -url string
        address to connect (default ":3000")
  -verbose
        verbose output
```

## Scripting

gws brings you ability to implement your tests logic in `.lua` scripts.
Please look at `scripts` folder in this repository to find an examples of scripting. 

## Why

`gws` is highly inspired by [wsd](https://github.com/alexanderGugel/wsd) and [iocat](https://github.com/moul/iocat). But in both
 tools I found not existing features that I was needed some how.
