# :sparkles: gws

> CLI tool for websocket testing

## Install

```shell
go get github.com/gobwas/gws
```

## Usage 

```shell
gws -u "ws://my.cool.address"
```


```
-H string
    headers list
    format:
            { pair[ ";" pair...] },
    pair:
            { key ":" value }
-l int
    limit of reconnections (default 1)
-u string
    websocket url
-v    verbosity
```

## Why

`gws` is highly inspired by [wsd](https://github.com/alexanderGugel/wsd) and (iocat)[https://github.com/moul/iocat]. But in both
 tools I found not existing features that I was needed some how.