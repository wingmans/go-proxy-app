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

	gatewayURL, err := url.Parse(envOrDefault(
		"GATEWAY_URL", "http://127.0.0.1:8090",
	))
	if err != nil {
		jsonLogger.Error("invalid gateway URL", slog.String("error", err.Error()))
		os.Exit(1)
	}

	client := &http.Client{
		Transport: &RateLimitRetryTransport{
			Transport: &GatewayTransport{
				GatewayURL: gatewayURL, Transport: http.DefaultTransport,
			},
			MaxRetries: 3,
			BaseDelay:  250 * time.Millisecond,
			Logger:     jsonLogger,
		},
		Timeout: 15 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	targetURL := envOrDefault("TARGET_URL", "https://example.com")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		jsonLogger.Error("invalid target URL", slog.String("error", err.Error()))
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		jsonLogger.Error("request failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		jsonLogger.Error("failed to read response", slog.String("error", err.Error()))
		os.Exit(1)
	}

	jsonLogger.Info("request completed",
		slog.String("target", targetURL),
		slog.Int("status", resp.StatusCode),
	)
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
