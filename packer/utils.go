package packer

import (
	"bytes"
	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
	"net/url"
	"path"
)

// Content-Type MIME of the most common data formats.
const (
	MIMEHTML = "text/html"
	MIMECSS  = "text/css"
)

func runJavaScript(wa *WebArchive) (err error) {
	res := wa.WebMainResource

	query, err := goquery.NewDocumentFromReader(bytes.NewReader(res.WebResourceData))
	if err != nil {
		return err
	}

	vm := otto.New()
	if _, err = vm.Run("var window = {};"); err != nil {
		return err
	}
	query.Find("script").Each(func(i int, selection *goquery.Selection) {
		if err != nil {
			return
		}
		//_, err = vm.Run(selection.Text())
	})

	return
}

func nextUrl(workQ chan string, topUrl, nextUrl string) {
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
