package main

import (
	"errors"
	"net/http"
	"net/url"
)

type GatewayTransport struct {
	GatewayURL *url.URL
	Transport  http.RoundTripper
}

func (t *GatewayTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t.GatewayURL == nil {
		return nil, errors.New("gateway URL is required")
	}
	if request.URL.Scheme != "https" {
		return nil, errors.New("gateway only supports HTTPS upstreams")
	}

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	forwarded := request.Clone(request.Context())
	forwarded.URL.Scheme = t.GatewayURL.Scheme
	forwarded.URL.Host = t.GatewayURL.Host
	forwarded.Host = request.URL.Host

	return transport.RoundTrip(forwarded)
}
