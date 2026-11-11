package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidationHandler(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "injected credential accepted",
			headers:    map[string]string{"X-Demo-Api-Key": "test-secret"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong credential rejected",
			headers:    map[string]string{"X-Demo-Api-Key": "wrong"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "leaked caller credential rejected",
			headers: map[string]string{
				"X-Demo-Api-Key": "test-secret",
				"Authorization":  "Bearer caller-secret",
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	handler := newValidationHandler("X-Demo-Api-Key", "test-secret")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/example", nil)
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}

			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tt.wantStatus)
			}
		})
	}
}
