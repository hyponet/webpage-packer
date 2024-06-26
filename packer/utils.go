package packer

import (
	"code.dny.dev/ssrf"
	"github.com/microcosm-cc/bluemonday"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Content-Type MIME of the most common data formats.
const (
	MIMEHTML = "text/html"
	MIMECSS  = "text/css"
)

func xssSanitize(bodyContent string) string {
	sanitized := bluemonday.UGCPolicy().Sanitize(bodyContent)
	return strings.TrimSpace(sanitized)
}

func nextUrl(workQ chan string, topUrl, nextUrl string) {
	if nextUrl == "" {
		return
	}

	if strings.HasPrefix(nextUrl, "data:") {
		return
	}

	topParsedUrl, err := url.Parse(topUrl)
	if err != nil {
		return
	}

	nextParsedUrl, err := url.Parse(nextUrl)
	if err != nil {
		u, err := url.Parse(topUrl)
		if err != nil {
			return
		}
		u.Path = path.Join(u.Path, nextUrl)
		nextParsedUrl = u
	}
	nextParsedUrl.Fragment = ""
	if nextParsedUrl.Scheme == "" {
		nextParsedUrl.Scheme = topParsedUrl.Scheme
	}
	workQ <- nextParsedUrl.String()
}

func newHttpClient(opt Option) (*http.Client, map[string]string) {

	dialer := &net.Dialer{
		Control:   ssrf.New().Safe,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if opt.EnablePrivateNet {
		dialer.Control = nil
	}

	cli := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: defaultTimeout,
	}
	if opt.Timeout > 0 {
		cli.Timeout = time.Second * time.Duration(opt.Timeout)
	}

	headers := map[string]string{"Referer": opt.URL}
	for k, v := range defaultHeaders {
		headers[k] = v
	}
	if len(opt.Headers) > 0 {
		for k, v := range opt.Headers {
			headers[k] = v
		}
	}
	return cli, headers
}
