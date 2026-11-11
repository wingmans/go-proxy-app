package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	proxyURL := strings.TrimRight(envOrDefault("AUTH_PROXY_URL", "http://127.0.0.1:8091"), "/")
	targetPath := envOrDefault("TARGET_PATH", "/v1/example")
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(proxyURL + targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "authorization proxy request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "authorization proxy returned %s: %s\n", resp.Status, body)
		os.Exit(1)
	}

	fmt.Printf("authorization proxy returned %s: %s", resp.Status, body)
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
