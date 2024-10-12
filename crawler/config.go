package crawler

import "net/url"

type Config struct {
	NumWorkers       int
	BaseURL          url.URL
	Resume           bool
	DownloadPathBase string
	Debug            bool
}
