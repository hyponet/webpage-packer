package packer

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"howett.net/plist"
)

var (
	defaultTimeout = time.Minute
	defaultHeaders = map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
		"Accept-Language": "en-us",
		"Accept-Encoding": "gzip, deflate",
	}
)

type WebArchive struct {
	WebMainResource WebResourceItem   `json:"WebMainResource"`
	WebSubresources []WebResourceItem `json:"WebSubresources"`
}

type WebResourceItem struct {
	WebResourceURL              string `json:"WebResourceURL"`
	WebResourceMIMEType         string `json:"WebResourceMIMEType"`
	WebResourceResponse         []byte `json:"WebResourceResponse,omitempty"`
	WebResourceData             []byte `json:"WebResourceData,omitempty"`
	WebResourceTextEncodingName string `json:"WebResourceTextEncodingName,omitempty"`
}

type webArchiver struct {
	workerQ  chan string
	resource *WebArchive
	seen     map[string]struct{}
	parallel int
	mux      sync.Mutex
}

func (w *webArchiver) Pack(ctx context.Context, opt Option) error {
	w.workerQ <- opt.URL

	wg := sync.WaitGroup{}
	errCh := make(chan error, 1)
	for i := 0; i < w.parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.workerRun(ctx, opt, errCh)
		}()
	}

	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		w.resource.WebMainResource = w.resource.WebSubresources[0]
	}

	if err := runJavaScript(w.resource); err != nil {
		return fmt.Errorf("run javascript failed: %s", err)
	}

	if opt.ClutterFree {
		err := MakeClutterFree(w.resource)
		if err != nil {
			return fmt.Errorf("make clustter free failed: %s", err)
		}
	}

	output, err := os.OpenFile(opt.Output, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0655)
	if err != nil {
		return fmt.Errorf("open output file failed: %s", err)
	}
	defer output.Close()

	encoder := plist.NewBinaryEncoder(output)
	err = encoder.Encode(w.resource)
	if err != nil {
		return fmt.Errorf("encode plist file error: %s", err)
	}

	return nil
}

func (w *webArchiver) workerRun(ctx context.Context, opt Option, errCh chan error) {
	cli := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   defaultTimeout,
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

	for {
		select {
		case urlStr, ok := <-w.workerQ:
			if !ok {
				return
			}
			w.mux.Lock()
			_, seen := w.seen[urlStr]
			w.mux.Unlock()
			if !seen {
				w.mux.Lock()
				w.seen[urlStr] = struct{}{}
				w.mux.Unlock()
				if err := w.loadWebPageFromUrl(ctx, cli, headers, urlStr); err != nil {
					select {
					case errCh <- err:
					default:

					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *webArchiver) loadWebPageFromUrl(ctx context.Context, cli *http.Client, headers map[string]string, urlStr string) error {
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return fmt.Errorf("build request with url %s error: %s", urlStr, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := cli.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("do request with url %s error: %s", urlStr, err)
	}
	defer func() {
		_, _ = ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("do request with url %s error: status code is %d", urlStr, resp.StatusCode)
	}

	var bodyReader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		bodyReader, err = gzip.NewReader(resp.Body)
	case "deflate":
		bodyReader = flate.NewReader(resp.Body)
	default:
		bodyReader = resp.Body
	}

	data, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return fmt.Errorf("read response body with url %s error: %s", urlStr, err)
	}

	contentType := resp.Header.Get("Content-Type")
	item := WebResourceItem{
		WebResourceURL:      urlStr,
		WebResourceMIMEType: contentType,
		WebResourceData:     data,
	}

	w.mux.Lock()
	w.resource.WebSubresources = append(w.resource.WebSubresources, item)
	w.mux.Unlock()

	switch {
	case strings.Contains(contentType, MIMEHTML):
		query, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("build doc query with url %s error: %s", urlStr, err)
		}

		query.Find("img").Each(func(i int, selection *goquery.Selection) {
			var (
				srcVal    string
				isExisted bool
			)
			srcVal, isExisted = selection.Attr("src")
			if isExisted {
				nextUrl(w.workerQ, urlStr, srcVal)
			}
			srcVal, isExisted = selection.Attr("data-src")
			if isExisted {
				nextUrl(w.workerQ, urlStr, srcVal)
			}
			srcVal, isExisted = selection.Attr("data-src-retina")
			if isExisted {
				nextUrl(w.workerQ, urlStr, srcVal)
			}
		})

		query.Find("script").Each(func(i int, selection *goquery.Selection) {
			srcVal, isExisted := selection.Attr("src")
			if isExisted {
				nextUrl(w.workerQ, urlStr, srcVal)
			}
		})

		query.Find("link").Each(func(i int, selection *goquery.Selection) {
			relVal, isExisted := selection.Attr("rel")
			if !isExisted || relVal != "stylesheet" {
				return
			}
			relVal, isExisted = selection.Attr("href")
			if isExisted {
				nextUrl(w.workerQ, urlStr, relVal)
			}
		})

		close(w.workerQ)

	case strings.Contains(contentType, MIMECSS):
		// TODO: parse @import url
	}

	return nil
}

func NewWebArchivePacker() Packer {
	return &webArchiver{
		workerQ:  make(chan string, 5),
		resource: &WebArchive{},
		seen:     map[string]struct{}{},
		parallel: 10,
	}
}
