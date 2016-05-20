package client

import (
	"flag"
	"fmt"
	"github.com/gobwas/gws/common"
	headersUtil "github.com/gobwas/gws/util/headers"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	uri     = flag.String("u", ":3000", "websocket url to connect")
	timeout = flag.String("t", "0", "client script run timeout")
)

const headerOrigin = "Origin"

type config struct {
	headers http.Header
	uri     *url.URL
	timeout time.Duration
}

func getConfig() (c config, err error) {
	headers, err := headersUtil.Parse(common.Headers)
	if err != nil {
		err = common.UsageError{err}
		return
	}

	uri, err := parseURL(*uri)
	if err != nil {
		err = common.UsageError{err}
		return
	}

	t, err := time.ParseDuration(*timeout)
	if err != nil {
		err = common.UsageError{err}
		return
	}

	c = config{
		uri:     uri,
		headers: fillOriginHeader(headers, uri),
		timeout: t,
	}

	return
}

func parseURL(rawURL string) (*url.URL, error) {
	// prevent false error on parsing url
	if strings.Index(rawURL, "://") == -1 {
		rawURL = fmt.Sprintf("ws://%s", rawURL)
	}
	uri, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if uri.Scheme == "" {
		uri.Scheme = "ws"
	}

	return uri, nil
}

func fillOriginHeader(headers http.Header, uri *url.URL) http.Header {
	// by default, set the same origin
	// to avoid same origin policy check on connections
	if headers.Get(headerOrigin) == "" {
		var s string
		switch uri.Scheme {
		case "wss":
			s = "https"
		default:
			s = "http"
		}
		orig := url.URL{
			Scheme: s,
			Host:   uri.Host,
		}
		headers.Set(headerOrigin, orig.String())
	}

	return headers
}
