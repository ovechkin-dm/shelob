package crawler

import (
	"strings"

	"golang.org/x/net/html"
)

type Parser interface {
	Parse(contents string) (*ParseResult, error)
}

type SimpleParser struct{}

type ParseResult struct {
	URLs []string
}

func (p *SimpleParser) Parse(contents string) (*ParseResult, error) {
	doc, err := html.Parse(strings.NewReader(contents))
	if err != nil {
		return nil, err
	}
	urls := extractUrls(doc)
	return &ParseResult{URLs: urls}, nil
}

func extractUrls(n *html.Node) []string {
	var urls []string
	if n.Type == html.ElementNode && strings.ToLower(n.Data) == "a" {
		for _, a := range n.Attr {
			if strings.ToLower(a.Key) == "href" {
				urls = append(urls, a.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		urls = append(urls, extractUrls(c)...)
	}
	return urls
}

func NewParser() *SimpleParser {
	return &SimpleParser{}
}
