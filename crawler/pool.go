package crawler

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	urllib "net/url"
)

type WorkerPool struct {
	cfg       *Config
	queue     Queue[string]
	parser    Parser
	repo      ContentRepository
	client    Client
	wg        sync.WaitGroup
	visited   sync.Map
	workersWG sync.WaitGroup
}

func (w *WorkerPool) Start(ctx context.Context) error {
	normalized, valid := w.normalizeURL(w.cfg.BaseURL.String())
	if !valid {
		return errors.New("invalid base url")
	}
	w.visited.Store(normalized.String(), struct{}{})
	w.wg.Add(1)
	w.queue.Put(normalized.String())
	w.runWorkers(ctx)
	w.WaitOn()
	w.queue.Close()
	return nil
}

func (w *WorkerPool) WaitOn() {
	done := make(chan struct{})
	go func() {
		w.workersWG.Wait()
		done <- struct{}{}
	}()
	go func() {
		w.wg.Wait()
		done <- struct{}{}
	}()
	<-done
}

func (w *WorkerPool) runWorkers(ctx context.Context) {
	w.workersWG.Add(w.cfg.NumWorkers)
	for i := 0; i < w.cfg.NumWorkers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					w.workersWG.Done()
					return
				case url, open := <-w.queue.Out():
					if !open {
						return
					}
					slog.DebugContext(ctx,
						"begin processing url",
						slog.String("url", url),
					)

					err := w.processUrl(ctx, url)

					if err != nil {
						slog.ErrorContext(ctx,
							"error processing url",
							slog.String("url", url),
							slog.String("error", err.Error()),
						)
					} else {
						slog.DebugContext(ctx,
							"done processing url",
							slog.String("url", url),
						)
					}
				}
			}
		}()
	}
}

func (w *WorkerPool) processUrl(ctx context.Context, rawURL string) error {
	var content string
	var err error
	defer w.wg.Done()
	url, err := urllib.Parse(rawURL)
	if err != nil {
		return err
	}
	filePath := w.toPath(url)
	if w.cfg.Resume && w.repo.Exists(filePath) {
		slog.DebugContext(ctx,
			"url already downloaded",
			slog.String("url", rawURL),
		)
		content, err = w.repo.GetData(filePath)
		if err != nil {
			return err
		}
	} else {
		content, err = w.client.Get(ctx, url)
		if err != nil {
			return err
		}
		err = w.repo.Save(filePath, content)
		if err != nil {
			return err
		}
	}

	parseResult, err := w.parser.Parse(content)
	if err != nil {
		return err
	}
	urls := w.processParsedLinks(parseResult)
	w.wg.Add(len(urls))
	for _, u := range urls {
		w.queue.Put(u.String())
	}
	return nil
}

func (w *WorkerPool) processParsedLinks(parseResult *ParseResult) []*urllib.URL {
	urls := make([]*urllib.URL, 0)
	for i := range parseResult.URLs {
		normalized, valid := w.normalizeURL(parseResult.URLs[i])
		if !valid {
			continue
		}
		_, exists := w.visited.LoadOrStore(normalized.String(), struct{}{})
		if exists {
			continue
		}
		urls = append(urls, normalized)
	}
	return urls
}

func (w *WorkerPool) toPath(url *urllib.URL) string {
	raw := w.cfg.DownloadPathBase + url.Host + "/" + url.Path
	raw = strings.Replace(raw, "//", "/", -1)
	splitted := strings.Split(raw, "/")
	if len(splitted) == 0 {
		return raw
	}
	last := splitted[len(splitted)-1]
	if strings.Contains(last, ".") {
		return raw
	}
	raw += "/index.html"
	raw = strings.Replace(raw, "//", "/", -1)
	return raw
}

func (w *WorkerPool) normalizeURL(rawURL string) (*urllib.URL, bool) {
	if len(rawURL) > 1 && rawURL[0:2] == "//" {
		return nil, false
	}
	if len(rawURL) > 0 && rawURL[0:1] == "/" {
		rawURL = w.cfg.BaseURL.String() + rawURL
	}
	if strings.Contains(rawURL, "mailto:") {
		return nil, false
	}
	if strings.Contains(rawURL, "..") {
		return nil, false
	}
	u, err := urllib.Parse(rawURL)
	if err != nil {
		return nil, false
	}
	if !strings.Contains(rawURL, w.cfg.BaseURL.String()) {
		return nil, false
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.Replace(u.Path, "//", "/", -1)
	return u, true
}

func NewWorkerPool(
	cfg *Config,
	queue Queue[string],
	parser Parser,
	contentRepository ContentRepository,
	client Client,
) *WorkerPool {
	return &WorkerPool{
		cfg:     cfg,
		queue:   queue,
		parser:  parser,
		repo:    contentRepository,
		client:  client,
		visited: sync.Map{},
	}
}
