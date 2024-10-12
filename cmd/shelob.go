package cmd

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ovechkin-dm/shelob/crawler"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func Execute() {
	config, err := parseConfig()
	if err != nil {
		log.Fatalf("failed to parse config: %v", err.Error())
	}
	initLogger(config)
	client := crawler.NewHTTPClient(config)
	parser := crawler.NewParser()
	queue := crawler.NewUnboundedQueue[string]()
	repo := crawler.NewFileSystemRepository()
	pool := crawler.NewWorkerPool(config, queue, parser, repo, client)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	go func() {
		<-c
		slog.Info("Shutting down due to user interrupt...")
		cancelFunc()
	}()

	err = pool.Start(ctx)
	if err != nil {
		slog.Error("failed to start worker pool: %v", "error", err.Error())
		os.Exit(1)
	}
}

func parseConfig() (*crawler.Config, error) {
	pflag.Int("workers", 1, "Number of workers to use")
	pflag.String("baseurl", "", "Base URL for the crawler")
	pflag.Bool("resume", false, "Resume previous download")
	pflag.String("downloadpath", "./downloads/", "Base path to download content")
	pflag.Bool("debug", false, "Enable debug mode")
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return nil, fmt.Errorf("failed to bind flags: %v", err.Error())
	}

	viper.SetEnvPrefix("app")
	viper.AutomaticEnv()

	viper.SetDefault("workers", 1)
	viper.SetDefault("baseurl", "")
	viper.SetDefault("resume", false)
	viper.SetDefault("downloadpath", "./downloads/")
	viper.SetDefault("debug", false)

	baseURLStr := viper.GetString("baseurl")
	baseURL, err := url.Parse(baseURLStr)
	if baseURLStr == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %s", baseURLStr)
	}
	if baseURL.Scheme == "" {
		return nil, fmt.Errorf("base URL must include scheme (http/https)")
	}
	downloadPathBase := viper.GetString("downloadpath")
	if downloadPathBase == "" {
		return nil, fmt.Errorf("download path is required")
	}
	if downloadPathBase[len(downloadPathBase)-1] != '/' {
		downloadPathBase += "/"
	}
	config := crawler.Config{
		NumWorkers:       viper.GetInt("workers"),
		BaseURL:          *baseURL,
		Resume:           viper.GetBool("resume"),
		DownloadPathBase: downloadPathBase,
		Debug:            viper.GetBool("debug"),
	}

	if config.Debug {
		log.Println("Debug mode enabled")
	}
	return &config, nil
}

func initLogger(cfg *crawler.Config) {
	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
