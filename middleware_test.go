package fetch

import (
	"net/http"
	"testing"
)

func TestSkip(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/path", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	executed := false
	middleware := Skip()
	handler := middleware(HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		executed = true
		return nil, nil
	}))

	handler.Handle(&http.Client{}, req)

	if !executed {
		t.Error("expected handler to be executed")
	}
}

func TestHandlerFunc(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/path", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	executed := false
	handler := HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		executed = true
		return &http.Response{StatusCode: 200}, nil
	})

	resp, err := handler.Handle(&http.Client{}, req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !executed {
		t.Error("expected handler to be executed")
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}
}

func TestMiddlewareChaining(t *testing.T) {
	// Test that middleware can wrap handlers and modify requests
	req, err := http.NewRequest("GET", "http://example.com/path", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Middleware that adds a header
	addHeaderMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Custom", "added")
			return next.Handle(client, req)
		})
	}

	finalHandler := HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		// Check the header was added
		if req.Header.Get("X-Custom") != "added" {
			t.Error("expected X-Custom header to be added")
		}
		return &http.Response{StatusCode: 200}, nil
	})

	// Chain the middleware
	handler := addHeaderMiddleware(finalHandler)

	_, err = handler.Handle(&http.Client{}, req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}

func TestMiddlewareCanShortCircuit(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/path", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	executed := false

	// Middleware that short-circuits the chain
	shortCircuitMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			// Don't call next handler, return immediately
			return &http.Response{StatusCode: 403}, nil
		})
	}

	finalHandler := HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		executed = true
		return &http.Response{StatusCode: 200}, nil
	})

	handler := shortCircuitMiddleware(finalHandler)

	resp, err := handler.Handle(&http.Client{}, req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if executed {
		t.Error("expected final handler not to be executed")
	}

	if resp.StatusCode != 403 {
		t.Errorf("expected status code 403, got %d", resp.StatusCode)
	}
}
