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
gws -u "ws://my.cool.address"
```

Run simple server and type response messages in terminal:

```shell
gws -l ":8888" -resp=prompt
```

Or just simple echo:

```shell
gws -l ":8888" -resp=echo
```

Usage info:

```
Usage of gws:
  -H string
        list of headers to be passed during handshake
        format:
                { pair[ ";" pair...] },
        pair:
                { key ":" value }
  -l string
        run ws server and listen this address
  -resp value
        how should server response on message (echo, mirror, prompt) (default mirror)
  -u string
        websocket server url
  -v    show additional debugging info
  -x int
        try to reconnect x times (default 1)
```

## Why

`gws` is highly inspired by [wsd](https://github.com/alexanderGugel/wsd) and [iocat](https://github.com/moul/iocat). But in both
 tools I found not existing features that I was needed some how.
