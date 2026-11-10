package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	proxyURL, _ := url.Parse(os.Getenv("PROXY_URL"))
	client := &http.Client{
		Transport: &RateLimitRetryTransport{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
			MaxRetries: 3,
			BaseDelay:  250 * time.Millisecond,
			Logger:     jsonLogger,
		},
		Timeout: 15 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.service-a.com/data", nil)

	resp, err := client.Do(req)
	if err != nil {
		jsonLogger.Error("request failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer resp.Body.Close()

	io.Copy(io.Discard, resp.Body)
}
