package web

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/liaozhai/crawler"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type response struct {
	body []byte
	urls []string
}

func (r response) Value() []byte {
	return r.body
}

func (r response) Keys() []string {
	return r.urls
}

func parse(base *url.URL, body []byte) ([]string, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	urls := []string{}
	s := make(map[string]struct{})
	for n := range doc.Descendants() {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			for _, a := range n.Attr {
				if a.Key == "href" {
					u, err := url.Parse(a.Val)
					if err != nil {
						break
					}
					switch {
					case u.IsAbs() && base.Host != u.Host || u.Fragment != "":
						break
					default:
						u.Scheme = base.Scheme
						u.Host = base.Host
						v := u.String()
						if _, ok := s[v]; !ok {
							s[v] = struct{}{}
							urls = append(urls, v)
						}
					}
					break
				}
			}
		}
	}
	return urls, nil
}

var transform = crawler.Transformer[string, []byte](func(base string) crawler.Interface[string, []byte] {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(base)
	if err != nil {
		return response{[]byte{}, []string{}}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response{[]byte{}, []string{}}
	}
	u, err := url.Parse(base)
	if err != nil {
		return response{[]byte{}, []string{}}
	}
	urls, err := parse(u, body)
	if err != nil {
		return response{[]byte{}, []string{}}
	}
	return response{body: body, urls: urls}
})

func Crawl(base string, depth int, out chan crawler.Result[string, []byte]) {
	crawler.Crawl(base, depth, transform, out)
}
