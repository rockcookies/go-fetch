package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestComposeOrder(t *testing.T) {
	var executionOrder []string

	// 创建一个模拟的 handler
	finalHandler := HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
		executionOrder = append(executionOrder, "handler")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		}, nil
	})

	// 创建三个中间件，每个都会记录执行顺序
	middleware1 := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			executionOrder = append(executionOrder, "middleware1-before")
			resp, err := next.Handle(client, req)
			executionOrder = append(executionOrder, "middleware1-after")
			return resp, err
		})
	}

	middleware2 := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			executionOrder = append(executionOrder, "middleware2-before")
			resp, err := next.Handle(client, req)
			executionOrder = append(executionOrder, "middleware2-after")
			return resp, err
		})
	}

	middleware3 := func(next Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			executionOrder = append(executionOrder, "middleware3-before")
			resp, err := next.Handle(client, req)
			executionOrder = append(executionOrder, "middleware3-after")
			return resp, err
		})
	}

	// compose(m1, m2, m3) 返回一个 Middleware，应用到 handler 后按照 m1 -> m2 -> m3 -> handler 的顺序执行
	composed := compose(middleware1, middleware2, middleware3)(finalHandler)

	// 执行请求
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := composed.Handle(&http.Client{}, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证执行顺序
	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"middleware3-before",
		"handler",
		"middleware3-after",
		"middleware2-after",
		"middleware1-after",
	}

	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d execution steps, got %d", len(expected), len(executionOrder))
	}

	for i, step := range expected {
		if executionOrder[i] != step {
			t.Errorf("step %d: expected %q, got %q", i, step, executionOrder[i])
		}
	}

	t.Logf("Execution order: %v", executionOrder)
}
