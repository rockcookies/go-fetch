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
				if d.client.Timeout != 30*time.Second {
					t.Errorf("expected default timeout 30s, got %v", d.client.Timeout)
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

	if d.client.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", d.client.Timeout)
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

func TestDispatcher_With(t *testing.T) {
	original := NewDispatcher(nil, Skip())

	// Test that With creates a new dispatcher
	newDispatcher := original.With(Skip())

	if newDispatcher == original {
		t.Error("expected With to create a new dispatcher")
	}

	if len(newDispatcher.middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(newDispatcher.middlewares))
	}

	if len(original.middlewares) != 1 {
		t.Error("expected original dispatcher to be unchanged")
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
