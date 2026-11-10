package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"time"
)

var (
	errMaxRateLimitRetries = errors.New("max rate limit retries")
	errUnexpectedRetryExit = errors.New("unexpected retry exit")
	errGenerateJitter      = errors.New("generate retry jitter")
)

type RateLimitRetryTransport struct {
	Transport  http.RoundTripper
	MaxRetries int
	BaseDelay  time.Duration
	Logger     *slog.Logger
}

func (t *RateLimitRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	maxRetries := t.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}

	baseDelay := t.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 200 * time.Millisecond
	}

	log := t.logger()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		var reqClone *http.Request

		if req.Body != nil && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}

			reqClone = req.Clone(req.Context())
			reqClone.Body = body
		} else {
			reqClone = req.Clone(req.Context())
		}

		resp, err := transport.RoundTrip(reqClone)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			if attempt > 0 {
				log.Info("rate limit retry succeeded",
					slog.String("host", req.URL.Host),
					slog.Int("total_attempts", attempt+1),
				)
			}

			return resp, nil
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if attempt == maxRetries {
			logMaxRetriesExceeded(log, req.URL.Host, maxRetries)

			return nil, fmt.Errorf("%w (%d) exceeded for host %s", errMaxRateLimitRetries, maxRetries, req.URL.Host)
		}

		jitter, err := retryJitter()
		if err != nil {
			return nil, err
		}

		backoff := (1<<attempt)*baseDelay + jitter

		log.Warn("rate limited (429) encountered, applying backoff",
			slog.String("host", req.URL.Host),
			slog.Int("attempt", attempt+1),
			slog.Duration("backoff_delay", backoff),
		)

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(backoff):
		}
	}

	return nil, errUnexpectedRetryExit
}

func logMaxRetriesExceeded(log *slog.Logger, host string, maxRetries int) {
	log.Error("max rate limit retries exceeded",
		slog.String("host", host),
		slog.Int("max_retries", maxRetries),
	)
}

func retryJitter() (time.Duration, error) {
	jitterMilliseconds, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errGenerateJitter, err)
	}

	return time.Duration(jitterMilliseconds.Int64()) * time.Millisecond, nil
}

func (t *RateLimitRetryTransport) logger() *slog.Logger {
	if t.Logger != nil {
		return t.Logger
	}

	return slog.Default()
}
