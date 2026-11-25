Yes—Caddy calculates and returns this automatically when using the mholt/caddy-ratelimit plugin, though stock Caddy does not include a built-in rate limiter.   1. Automatic Retry-After via mholt/caddy-ratelimitBecause mholt/caddy-ratelimit tracks timestamps in a sliding-window ring buffer rather than a stateless counter, it knows the exact timestamp of the oldest request in the current window.   When a request exceeds your configured quota (events per window), the module:Rejects the request with HTTP 429 Too Many Requests.   Automatically calculates the remaining time until the oldest event falls out of the sliding window and sets the standard Retry-After response header (in seconds).   Optionally adds random jitter to the calculated Retry-After value so multiple callers do not retry at the exact same millisecond.   Example Caddyfile configuration for a 10 RPS quota:Code snippet{
	order rate_limit before reverse_proxy
}

:8080 {
	rate_limit {
		zone outbound_quota {
			key static
			events 10
			window 1s
		}
		jitter 0.1
	}

	reverse_proxy https://api.example.com
}
2. Customizing the 429 Response BodyIf your caller expects a JSON payload instead of only reading the Retry-After header, you can intercept the 429 error in handle_errors and forward the computed header value ({http.response.header.Retry-After}) into the response body:Code snippet	handle_errors {
		@ratelimit expression `{err.status_code} == 429`
		handle @ratelimit {
			header Content-Type application/json
			respond `{"error":"rate_limited","retry_after_seconds":{http.response.header.Retry-After}}` 429
		}
	}
Key Caveat: Sub-Second PrecisionThe HTTP Retry-After header specification uses whole seconds (1, 2, etc.). If your quota is 10 requests per 1s and a caller hits the limit 100ms into the window, Retry-After rounds up to 1 second rather than 900ms. If your caller needs sub-second precision, returning a millisecond wait value from your Go proxy's limiter.Reserve().Delay() gives tighter pacing than Caddy's standard Retry-After header