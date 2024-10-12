package crawler

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	. "github.com/ovechkin-dm/mockio/mock"
)

func TestHappyPath(t *testing.T) {
	SetUp(t)
	queue := NewUnboundedQueue[string]()
	parser := Mock[Parser]()
	repo := Mock[ContentRepository]()
	client := Mock[Client]()
	ctx := context.Background()
	baseUrl, _ := url.Parse("https://example.com")
	pageUrl, _ := url.Parse("https://example.com/page1")
	link := fmt.Sprintf(`<a href="%s">Page 1</a>`, pageUrl.String())
	parseResult := &ParseResult{
		URLs: []string{pageUrl.String()},
	}
	cfg := &Config{
		NumWorkers:       1,
		BaseURL:          *baseUrl,
		Resume:           true,
		DownloadPathBase: "./downloads/",
		Debug:            true,
	}

	pool := NewWorkerPool(cfg, queue, parser, repo, client)

	WhenDouble(client.Get(AnyContext(), Any[*url.URL]())).
		ThenReturn(link, nil).
		Verify(Times(2))

	WhenDouble(parser.Parse(AnyString())).
		ThenReturn(parseResult, nil).
		Verify(Times(2))

	WhenSingle(repo.Exists(AnyString())).
		ThenReturn(false).
		Verify(Times(2))

	WhenSingle(repo.Save("./downloads/example.com/index.html", link)).
		ThenReturn(nil).
		Verify(Once())

	WhenSingle(repo.Save("./downloads/example.com/page1/index.html", link)).
		ThenReturn(nil).
		Verify(Once())

	err := pool.Start(ctx)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	VerifyNoMoreInteractions(repo)
	VerifyNoMoreInteractions(parser)
	VerifyNoMoreInteractions(client)
}
