// Package config brings common configuration flags and utils.
package config

import (
	"flag"
	"fmt"
	"github.com/gobwas/gws/util/headers"
	headersUtil "github.com/gobwas/gws/util/headers"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var Verbose bool
var HeaderList *headerList
var Addr string
var URI string
var Stat time.Duration

func init() {
	HeaderList = newHeaderList()
	// BoolVar and StringVar are used here just for reading them
	// from other packages with pure common.{Verbose|Headers} (without *)
	flag.BoolVar(&Verbose, "verbose", false, "verbose output")
	flag.StringVar(&Addr, "listen", ":3000", "address to listen")
	flag.StringVar(&URI, "url", ":3000", "address to connect")
	flag.DurationVar(&Stat, "statd", time.Second, "server statistics dump interval")
	flag.Var(HeaderList, "header", fmt.Sprintf("allows to specify list of headers to be passed during handshake (both in client or server)\n\tformat:\n\t\t{ key %s value }", headers.AssignmentOperator))
	flag.Var(HeaderList, "H", fmt.Sprintf("allows to specify list of headers to be passed during handshake (both in client or server)\n\tformat:\n\t\t{ key %s value }", headers.AssignmentOperator))
}

type headerList struct {
	list http.Header
}

func newHeaderList() *headerList {
	return &headerList{
		list: make(http.Header),
	}
}

func (h *headerList) Set(s string) error {
	k, v, err := headersUtil.ParseOne(s)
	if err != nil {
		return err
	}
	h.list.Add(k, v)
	return nil
}

func (h *headerList) String() string {
	return fmt.Sprintf("%v", h.list)
}

const headerOrigin = "Origin"

type Config struct {
	Addr     string
	URI      string
	Headers  http.Header
	StatDump time.Duration
}

func Parse() (c Config, err error) {
	headers := HeaderList.list
	uri, err := parseURL(URI)
	if err != nil {
		return
	}

	c = Config{
		Addr:     Addr,
		URI:      uri.String(),
		Headers:  fillOriginHeader(headers, uri),
		StatDump: Stat,
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
