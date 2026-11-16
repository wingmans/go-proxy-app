package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGatewayTransport(t *testing.T) {
	var received *http.Request
	gateway := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter, request *http.Request,
	) {
		received = request.Clone(request.Context())
		writer.WriteHeader(http.StatusOK)
	}))
	defer gateway.Close()

	gatewayURL, err := url.Parse(gateway.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &GatewayTransport{
		GatewayURL: gatewayURL, Transport: http.DefaultTransport,
	}}
	request, err := http.NewRequestWithContext(context.Background(),
		http.MethodGet, "https://api.example.com/path?value=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("User-Agent", "gateway-test")

	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	if received == nil {
		t.Fatal("gateway did not receive a request")
	}
	if received.Host != "api.example.com" {
		t.Errorf("got Host %q, want api.example.com", received.Host)
	}
	if received.URL.RequestURI() != "/path?value=1" {
		t.Errorf("got request URI %q", received.URL.RequestURI())
	}
	if received.Header.Get("User-Agent") != "gateway-test" {
		t.Errorf("got User-Agent %q", received.Header.Get("User-Agent"))
	}
}

func TestGatewayTransportRejectsHTTPUpstream(t *testing.T) {
	gatewayURL, err := url.Parse("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}

	transport := &GatewayTransport{GatewayURL: gatewayURL}
	request, err := http.NewRequestWithContext(context.Background(),
		http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = transport.RoundTrip(request)
	if err == nil || !strings.Contains(err.Error(), "only supports HTTPS") {
		t.Fatalf("got error %v", err)
	}
}
