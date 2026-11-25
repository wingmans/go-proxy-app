package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestRateLimitRetryTransport_RetryHints(t *testing.T) {
	for _, test := range []struct {
		name         string
		milliseconds string
		retryAfter   string
		want         time.Duration
	}{
		{name: "milliseconds take precedence", milliseconds: "213", retryAfter: "1", want: 213 * time.Millisecond},
		{name: "invalid milliseconds fall back to seconds", milliseconds: "-1", retryAfter: "2", want: 2 * time.Second},
		{name: "overflowing milliseconds fall back to seconds", milliseconds: "9223372036854775807", retryAfter: "2", want: 2 * time.Second},
		{name: "HTTP date", retryAfter: "Sat, 01 Jan 2000 00:00:03 GMT", want: 3 * time.Second},
		{name: "past HTTP date", retryAfter: "Fri, 31 Dec 1999 23:59:59 GMT"},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				attempts := 0
				transport := &RateLimitRetryTransport{
					BaseDelay: 10 * time.Second,
					Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						attempts++
						resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}
						if attempts == 1 {
							resp.StatusCode = http.StatusTooManyRequests
							resp.Header.Set("Retry-After-Ms", test.milliseconds)
							resp.Header.Set("Retry-After", test.retryAfter)
						}
						return resp, nil
					}),
				}
				req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
				resp, err := transport.RoundTrip(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				if attempts != 2 || resp.StatusCode != http.StatusOK {
					t.Fatalf("got %d attempts and HTTP %d, want 2 attempts and HTTP 200", attempts, resp.StatusCode)
				}
				if elapsed := time.Since(start); elapsed != test.want {
					t.Errorf("retried after %v, want %v", elapsed, test.want)
				}
			})
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRateLimitRetryTransport_SuccessAfterRetry(t *testing.T) {
	var attempts atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentAttempt := attempts.Add(1)
		if currentAttempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer ts.Close()

	client := &http.Client{
		Transport: &RateLimitRetryTransport{
			Transport:  http.DefaultTransport,
			MaxRetries: 3,
			BaseDelay:  1 * time.Millisecond,
		},
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(bodyBytes) != "Success" {
		t.Errorf("expected status 200 and 'Success', got status %d body %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestRateLimitRetryTransport_MaxRetriesExceeded(t *testing.T) {
	var attempts atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	client := &http.Client{
		Transport: &RateLimitRetryTransport{
			Transport:  http.DefaultTransport,
			MaxRetries: 2,
			BaseDelay:  1 * time.Millisecond,
		},
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)

	resp, err := client.Do(req)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}

		t.Fatalf("expected error due to max retries exceeded, got nil")
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 total attempts (1 initial + 2 retries), got %d", got)
	}
}

func TestRateLimitRetryTransport_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	client := &http.Client{
		Transport: &RateLimitRetryTransport{
			Transport:  http.DefaultTransport,
			MaxRetries: 5,
			BaseDelay:  50 * time.Millisecond,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
}

func TestRateLimitRetryTransport_PostPreservesBody(t *testing.T) {
	var (
		attempts       atomic.Int32
		receivedBodies []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentAttempt := attempts.Add(1)
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBodies = append(receivedBodies, string(bodyBytes))

		if currentAttempt == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &http.Client{
		Transport: &RateLimitRetryTransport{
			Transport:  http.DefaultTransport,
			MaxRetries: 2,
			BaseDelay:  1 * time.Millisecond,
		},
	}

	payload := "test-payload"
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL, bytes.NewBufferString(payload))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBufferString(payload)), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if len(receivedBodies) != 2 || receivedBodies[0] != payload || receivedBodies[1] != payload {
		t.Errorf("body payload was not correctly preserved across retries: %v", receivedBodies)
	}
}
