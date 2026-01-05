package fetch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRequest_Use(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	if len(req.middlewares) != 0 {
		t.Error("expected empty middlewares initially")
	}

	req.Use(Skip())

	if len(req.middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(req.middlewares))
	}

	// Test method chaining
	req2 := req.Use(Skip(), Skip())

	if req2 != req {
		t.Error("expected Use to return the same request for chaining")
	}

	if len(req.middlewares) != 3 {
		t.Errorf("expected 3 middlewares, got %d", len(req.middlewares))
	}
}

func TestRequest_Clone(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest().Use(Skip())

	cloned := req.Clone()

	if cloned == req {
		t.Error("expected Clone to create a new request")
	}

	if cloned.dispatcher != req.dispatcher {
		t.Error("expected cloned request to reference same dispatcher")
	}

	if len(cloned.middlewares) != len(req.middlewares) {
		t.Error("expected cloned middlewares to have same length")
	}

	// Modify cloned middlewares shouldn't affect original
	cloned.Use(Skip())

	if len(req.middlewares) != 1 {
		t.Error("expected original request middlewares unchanged")
	}

	if len(cloned.middlewares) != 2 {
		t.Error("expected cloned request to have added middleware")
	}
}

func TestRequest_Body(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	reader := strings.NewReader("test body")
	req.Body(reader)

	if len(req.middlewares) != 1 {
		t.Error("expected Body to add middleware")
	}
}

func TestRequest_JSON(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	data := map[string]string{"key": "value"}
	req.JSON(data)

	if len(req.middlewares) != 1 {
		t.Error("expected JSON to add middleware")
	}
}

func TestRequest_Form(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	form := url.Values{}
	form.Set("key", "value")
	req.Form(form)

	if len(req.middlewares) != 1 {
		t.Error("expected Form to add middleware")
	}
}

func TestRequest_Header(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	req.Header(func(h http.Header) {
		h.Set("X-Custom", "test")
	})

	if len(req.middlewares) != 1 {
		t.Error("expected Header to add middleware")
	}
}

func TestRequest_Query(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	req.Query(func(q url.Values) {
		q.Set("key", "value")
	})

	if len(req.middlewares) != 1 {
		t.Error("expected Query to add middleware")
	}
}

func TestRequest_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	}))
	defer server.Close()

	d := NewDispatcher(nil)
	resp := d.NewRequest().Send("GET", server.URL)

	if resp.Error != nil {
		t.Fatalf("Send returned error: %v", resp.Error)
	}

	if resp.RawResponse.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.RawResponse.StatusCode)
	}

	body := resp.String()
	if body != "response body" {
		t.Errorf("expected body %q, got %q", "response body", body)
	}
}

func TestRequest_SendContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context was set
		if r.Context() == context.Background() {
			t.Error("expected custom context")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(nil)
	ctx := context.WithValue(context.Background(), "test", "value")
	resp := d.NewRequest().SendContext(ctx, "GET", server.URL)

	if resp.Error != nil {
		t.Fatalf("SendContext returned error: %v", resp.Error)
	}

	if resp.RawResponse.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.RawResponse.StatusCode)
	}
}

func TestRequest_SendContext_NilContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(nil)
	resp := d.NewRequest().SendContext(nil, "GET", server.URL)

	if resp.Error != nil {
		t.Fatalf("SendContext with nil context returned error: %v", resp.Error)
	}
}

func TestRequest_Send_InvalidURL(t *testing.T) {
	d := NewDispatcher(nil)
	resp := d.NewRequest().Send("GET", "://invalid-url")

	if resp.Error == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestRequest_HTTPMethods(t *testing.T) {
	methods := []struct {
		name   string
		method string
		fn     func(*Request, string) *Response
	}{
		{"GET", "GET", func(r *Request, url string) *Response { return r.Get(url) }},
		{"POST", "POST", func(r *Request, url string) *Response { return r.Post(url) }},
		{"PUT", "PUT", func(r *Request, url string) *Response { return r.Put(url) }},
		{"PATCH", "PATCH", func(r *Request, url string) *Response { return r.Patch(url) }},
		{"DELETE", "DELETE", func(r *Request, url string) *Response { return r.Delete(url) }},
		{"HEAD", "HEAD", func(r *Request, url string) *Response { return r.Head(url) }},
		{"OPTIONS", "OPTIONS", func(r *Request, url string) *Response { return r.Options(url) }},
		{"TRACE", "TRACE", func(r *Request, url string) *Response { return r.Trace(url) }},
	}

	for _, tt := range methods {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("expected method %q, got %q", tt.method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			d := NewDispatcher(nil)
			resp := tt.fn(d.NewRequest(), server.URL)

			if resp.Error != nil {
				t.Fatalf("%s returned error: %v", tt.name, resp.Error)
			}

			if resp.RawResponse.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.RawResponse.StatusCode)
			}
		})
	}
}

func TestRequest_HTTPMethodsContext(t *testing.T) {
	methods := []struct {
		name   string
		method string
		fn     func(*Request, context.Context, string) *Response
	}{
		{"GetContext", "GET", func(r *Request, ctx context.Context, url string) *Response { return r.GetContext(ctx, url) }},
		{"PostContext", "POST", func(r *Request, ctx context.Context, url string) *Response { return r.PostContext(ctx, url) }},
		{"PutContext", "PUT", func(r *Request, ctx context.Context, url string) *Response { return r.PutContext(ctx, url) }},
		{"PatchContext", "PATCH", func(r *Request, ctx context.Context, url string) *Response { return r.PatchContext(ctx, url) }},
		{"DeleteContext", "DELETE", func(r *Request, ctx context.Context, url string) *Response { return r.DeleteContext(ctx, url) }},
		{"HeadContext", "HEAD", func(r *Request, ctx context.Context, url string) *Response { return r.HeadContext(ctx, url) }},
		{"OptionsContext", "OPTIONS", func(r *Request, ctx context.Context, url string) *Response { return r.OptionsContext(ctx, url) }},
		{"TraceContext", "TRACE", func(r *Request, ctx context.Context, url string) *Response { return r.TraceContext(ctx, url) }},
	}

	for _, tt := range methods {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("expected method %q, got %q", tt.method, r.Method)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			d := NewDispatcher(nil)
			ctx := context.Background()
			resp := tt.fn(d.NewRequest(), ctx, server.URL)

			if resp.Error != nil {
				t.Fatalf("%s returned error: %v", tt.name, resp.Error)
			}

			if resp.RawResponse.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.RawResponse.StatusCode)
			}
		})
	}
}

func TestRequest_Do(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if middleware was applied
		if r.Header.Get("X-Custom") != "test" {
			t.Error("expected middleware to add header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(nil)
	req := d.NewRequest()

	// Add middleware
	req.Use(func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Custom", "test")
			return next.Handle(client, req)
		})
	})

	httpReq, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create HTTP request: %v", err)
	}

	resp, err := req.Do(httpReq)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequest_MiddlewareChaining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	d := NewDispatcher(nil)

	resp := d.NewRequest().
		AddHeaderKV("X-Custom-1", "value1").
		AddHeaderKV("X-Custom-2", "value2").
		SetQueryKV("param", "value").
		Get(server.URL)

	if resp.Error != nil {
		t.Fatalf("chained request returned error: %v", resp.Error)
	}

	if resp.RawResponse.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.RawResponse.StatusCode)
	}
}

func TestRequest_BodyGet(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	called := false
	req.BodyGet(func() (io.Reader, error) {
		called = true
		return strings.NewReader("lazy body"), nil
	})

	if called {
		t.Error("expected BodyGet not to call the function immediately")
	}

	if len(req.middlewares) != 1 {
		t.Error("expected BodyGet to add middleware")
	}
}

func TestRequest_BodyGetBytes(t *testing.T) {
	d := NewDispatcher(nil)
	req := d.NewRequest()

	called := false
	req.BodyGetBytes(func() ([]byte, error) {
		called = true
		return []byte("lazy bytes"), nil
	})

	if called {
		t.Error("expected BodyGetBytes not to call the function immediately")
	}

	if len(req.middlewares) != 1 {
		t.Error("expected BodyGetBytes to add middleware")
	}
}
