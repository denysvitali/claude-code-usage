package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvidersEndpoint(t *testing.T) {
	s := NewServer(&Config{Host: "127.0.0.1", Port: 0})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/providers", nil)
	res := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	var providers []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &providers); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestUnknownRoute(t *testing.T) {
	s := NewServer(&Config{Host: "127.0.0.1", Port: 0})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", nil)
	res := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
}
