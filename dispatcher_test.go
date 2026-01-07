package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewDispatcher(t *testing.T) {
	tests := []struct {
		name                string
		client              *http.Client
		middlewares         []Middleware
		expectDefaultClient bool
	}{
		{
			name:                "with nil client creates default",
			client:              nil,
			middlewares:         []Middleware{},
			expectDefaultClient: true,
		},
		{
			name: "with custom client",
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
			middlewares:         []Middleware{},
			expectDefaultClient: false,
		},
		{
			name:   "with middleware",
			client: nil,
			middlewares: []Middleware{
				Skip(),
			},
			expectDefaultClient: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher(tt.client, tt.middlewares...)

			if d == nil {
				t.Fatal("expected dispatcher to be created")
			}

			if d.client == nil {
				t.Error("expected client to be set")
			}

			if tt.expectDefaultClient {
				if d.client.Timeout != 0 {
					t.Errorf("expected default timeout to be 0 (no timeout), got %v", d.client.Timeout)
				}
			}

			if len(d.middlewares) != len(tt.middlewares) {
				t.Errorf("expected %d middlewares, got %d", len(tt.middlewares), len(d.middlewares))
			}
		})
	}
}

func TestNewDispatcherWithTransport(t *testing.T) {
	customTransport := http.DefaultTransport

	d := NewDispatcherWithTransport(customTransport)

	if d == nil {
		t.Fatal("expected dispatcher to be created")
	}

	if d.client == nil {
		t.Error("expected client to be set")
	}

	if d.client.Transport != customTransport {
		t.Error("expected custom transport to be used")
	}

	if d.client.Timeout != 0 {
		t.Errorf("expected default timeout to be 0 (no timeout), got %v", d.client.Timeout)
	}
}

func TestDispatcher_Client(t *testing.T) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	d := NewDispatcher(client)

	if d.Client() != client {
		t.Error("expected Client() to return the same client")
	}
}

func TestDispatcher_SetClient(t *testing.T) {
	d := NewDispatcher(nil)

	newClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	d.SetClient(newClient)

	if d.client != newClient {
		t.Error("expected client to be updated")
	}

	// Test nil client does nothing
	d.SetClient(nil)
	if d.client != newClient {
		t.Error("expected client to remain unchanged when setting nil")
	}
}

func TestDispatcher_Middlewares(t *testing.T) {
	middlewares := []Middleware{Skip(), Skip()}
	d := NewDispatcher(nil, middlewares...)

	mws := d.Middlewares()
	if len(mws) != len(middlewares) {
		t.Errorf("expected %d middlewares, got %d", len(middlewares), len(mws))
	}
}

func TestDispatcher_SetMiddlewares(t *testing.T) {
	d := NewDispatcher(nil)

	newMiddlewares := []Middleware{Skip(), Skip()}
	d.SetMiddlewares(newMiddlewares...)

	if len(d.middlewares) != len(newMiddlewares) {
		t.Errorf("expected %d middlewares, got %d", len(newMiddlewares), len(d.middlewares))
	}
}

func TestDispatcher_Use(t *testing.T) {
	d := NewDispatcher(nil, Skip())

	if len(d.middlewares) != 1 {
		t.Errorf("expected 1 middleware initially, got %d", len(d.middlewares))
	}

	// Test that Use appends middleware
	d.Use(Skip(), Skip())

	if len(d.middlewares) != 3 {
		t.Errorf("expected 3 middlewares after Use, got %d", len(d.middlewares))
	}
}

func TestDispatcher_Clone(t *testing.T) {
	original := NewDispatcher(&http.Client{Timeout: 10 * time.Second}, Skip())

	cloned := original.Clone()

	if cloned == original {
		t.Error("expected Clone to create a new dispatcher")
	}

	if cloned.client == original.client {
		t.Error("expected cloned client to be different object")
	}

	if cloned.client.Timeout != original.client.Timeout {
		t.Error("expected cloned client to have same timeout")
	}

	if len(cloned.middlewares) != len(original.middlewares) {
		t.Error("expected cloned middlewares to have same length")
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if middleware added the header
		if r.Header.Get("X-Custom") != "test" {
			t.Error("expected X-Custom header to be added by middleware")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Middleware that adds a header
	addHeaderMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Custom", "test")
			return next.Handle(client, req)
		})
	}

	d := NewDispatcher(nil, addHeaderMiddleware)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := d.Dispatch(req)
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDispatcher_Dispatch_WithAdditionalMiddleware(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check both middleware added headers
		if r.Header.Get("X-Dispatcher") != "dispatcher" {
			t.Error("expected X-Dispatcher header from dispatcher middleware")
		}
		if r.Header.Get("X-Additional") != "additional" {
			t.Error("expected X-Additional header from additional middleware")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dispatcherMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Dispatcher", "dispatcher")
			return next.Handle(client, req)
		})
	}

	additionalMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Additional", "additional")
			return next.Handle(client, req)
		})
	}

	d := NewDispatcher(nil, dispatcherMiddleware)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := d.Dispatch(req, additionalMiddleware)
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDispatcher_NewRequest(t *testing.T) {
	d := NewDispatcher(nil)

	req := d.NewRequest()

	if req == nil {
		t.Fatal("expected request to be created")
	}

	if req.dispatcher != d {
		t.Error("expected request to reference dispatcher")
	}
}

func TestDispatcher_R(t *testing.T) {
	d := NewDispatcher(nil)

	req := d.R()

	if req == nil {
		t.Fatal("expected request to be created")
	}

	if req.dispatcher != d {
		t.Error("expected request to reference dispatcher")
	}
}

// TestDispatcher_MiddlewareExecutionOrder verifies the execution order of middlewares:
// d.middlewares -> middlewares (per-request) -> d.coreMiddlewares
func TestDispatcher_MiddlewareExecutionOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Track execution order using a shared slice
	var executionOrder []string

	// Create middleware that records when it executes
	createTrackingMiddleware := func(name string) Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				executionOrder = append(executionOrder, name)
				return next.Handle(client, req)
			})
		}
	}

	// Set up dispatcher with all three middleware types
	d := NewDispatcher(nil)
	d.Use(createTrackingMiddleware("dispatcher-1"))
	d.Use(createTrackingMiddleware("dispatcher-2"))
	d.UseCore(createTrackingMiddleware("core-1"))
	d.UseCore(createTrackingMiddleware("core-2"))

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Execute with per-request middlewares
	_, err = d.Dispatch(req,
		createTrackingMiddleware("request-1"),
		createTrackingMiddleware("request-2"),
	)
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}

	// Verify execution order: dispatcher middlewares -> request middlewares -> core middlewares
	expected := []string{
		"dispatcher-1",
		"dispatcher-2",
		"request-1",
		"request-2",
		"core-1",
		"core-2",
	}

	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d middleware executions, got %d", len(expected), len(executionOrder))
	}

	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("execution order at position %d: expected %s, got %s", i, exp, executionOrder[i])
		}
	}
}

// TestDispatcher_CoreMiddlewaresGetters tests CoreMiddlewares getter and setter
func TestDispatcher_CoreMiddlewares(t *testing.T) {
	d := NewDispatcher(nil)

	if len(d.CoreMiddlewares()) != 0 {
		t.Errorf("expected 0 core middlewares initially, got %d", len(d.CoreMiddlewares()))
	}

	coreMiddlewares := []Middleware{Skip(), Skip()}
	d.SetCoreMiddlewares(coreMiddlewares...)

	if len(d.CoreMiddlewares()) != len(coreMiddlewares) {
		t.Errorf("expected %d core middlewares, got %d", len(coreMiddlewares), len(d.CoreMiddlewares()))
	}
}

// TestDispatcher_UseCore tests UseCore method
func TestDispatcher_UseCore(t *testing.T) {
	d := NewDispatcher(nil)

	if len(d.coreMiddlewares) != 0 {
		t.Errorf("expected 0 core middlewares initially, got %d", len(d.coreMiddlewares))
	}

	d.UseCore(Skip())

	if len(d.coreMiddlewares) != 1 {
		t.Errorf("expected 1 core middleware after UseCore, got %d", len(d.coreMiddlewares))
	}

	d.UseCore(Skip(), Skip())

	if len(d.coreMiddlewares) != 3 {
		t.Errorf("expected 3 core middlewares after second UseCore, got %d", len(d.coreMiddlewares))
	}
}

// TestDispatcher_Clone_WithCoreMiddlewares ensures Clone copies core middlewares
func TestDispatcher_Clone_WithCoreMiddlewares(t *testing.T) {
	original := NewDispatcher(&http.Client{Timeout: 10 * time.Second})
	original.Use(Skip())
	original.UseCore(Skip(), Skip())

	cloned := original.Clone()

	if cloned == original {
		t.Error("expected Clone to create a new dispatcher")
	}

	if len(cloned.coreMiddlewares) != len(original.coreMiddlewares) {
		t.Errorf("expected cloned core middlewares length to match original: expected %d, got %d",
			len(original.coreMiddlewares), len(cloned.coreMiddlewares))
	}

	// Verify that modifying cloned middlewares doesn't affect original
	cloned.UseCore(Skip())

	if len(cloned.coreMiddlewares) == len(original.coreMiddlewares) {
		t.Error("expected cloned middlewares to be independent from original")
	}
}

// TestDispatcher_MiddlewareLayering tests that middlewares wrap correctly
func TestDispatcher_MiddlewareLayering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that all headers are present (proving all middlewares executed)
		headers := []string{"X-Dispatcher", "X-Request", "X-Core"}
		for _, h := range headers {
			if r.Header.Get(h) == "" {
				t.Errorf("expected header %s to be set", h)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	createHeaderMiddleware := func(headerName, headerValue string) Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
				req.Header.Set(headerName, headerValue)
				return next.Handle(client, req)
			})
		}
	}

	d := NewDispatcher(nil)
	d.Use(createHeaderMiddleware("X-Dispatcher", "dispatcher"))
	d.UseCore(createHeaderMiddleware("X-Core", "core"))

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := d.Dispatch(req, createHeaderMiddleware("X-Request", "request"))
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
