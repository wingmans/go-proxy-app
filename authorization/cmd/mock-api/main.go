package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const listenAddress = ":8080"

type validationResponse struct {
	Authenticated bool   `json:"authenticated"`
	Path          string `json:"path,omitempty"`
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	expectedHeader := envOrDefault("EXPECTED_AUTH_HEADER", "X-Demo-Api-Key")
	expectedValue := envOrDefault("EXPECTED_AUTH_VALUE", "test-secret")

	server := &http.Server{
		Addr:              listenAddress,
		Handler:           newValidationHandler(expectedHeader, expectedValue),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("mock API listening on %s", listenAddress)
	log.Fatal(server.ListenAndServe())
}

func newValidationHandler(expectedHeader, expectedValue string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Header.Get(expectedHeader) != expectedValue || hasLeakedCredential(r, expectedHeader) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(validationResponse{Authenticated: false})
			return
		}

		_ = json.NewEncoder(w).Encode(validationResponse{
			Authenticated: true,
			Path:          r.URL.Path,
		})
	})

	return mux
}

func hasLeakedCredential(r *http.Request, expectedHeader string) bool {
	for _, header := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
		if strings.EqualFold(header, expectedHeader) {
			continue
		}

		if r.Header.Get(header) != "" {
			return true
		}
	}

	return false
}

func runHealthcheck() {
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/healthz")
	if err != nil {
		log.Printf("healthcheck failed: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("healthcheck returned status %d", resp.StatusCode)
		os.Exit(1)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
