package dump

import (
	"net/http"
	"net/url"
	"testing"
)

func makeReq(method, path string) *http.Request {
	return &http.Request{
		Method: method,
		URL:    &url.URL{Path: path},
		Header: http.Header{},
	}
}

func makeResp(status int) *http.Response {
	return &http.Response{StatusCode: status}
}

func TestURLFilter(t *testing.T) {
	f := URLFilter("/api/", "/internal/")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "api_prefix", path: "/api/users", want: true},
		{name: "internal_prefix", path: "/internal/health", want: true},
		{name: "no_match", path: "/public", want: false},
		{name: "root", path: "/", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeReq("GET", tt.path)
			if got := f.Match(req, nil); got != tt.want {
				t.Errorf("URLFilter(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestMethodFilter(t *testing.T) {
	f := MethodFilter("POST", "put")

	tests := []struct {
		name   string
		method string
		want   bool
	}{
		{name: "post", method: "POST", want: true},
		{name: "put_normalized", method: "PUT", want: true},
		{name: "get", method: "GET", want: false},
		{name: "delete", method: "DELETE", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeReq(tt.method, "/")
			if got := f.Match(req, nil); got != tt.want {
				t.Errorf("MethodFilter(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestStatusFilter(t *testing.T) {
	f := StatusFilter([2]int{400, 499}, [2]int{500, 599})

	tests := []struct {
		name string
		resp *http.Response
		want bool
	}{
		{name: "ok_200", resp: makeResp(200), want: false},
		{name: "client_400", resp: makeResp(400), want: true},
		{name: "client_404", resp: makeResp(404), want: true},
		{name: "server_500", resp: makeResp(500), want: true},
		{name: "nil_resp", resp: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := f.Match(makeReq("GET", "/"), tt.resp); got != tt.want {
				t.Errorf("StatusFilter = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeaderFilter(t *testing.T) {
	f := HeaderFilter("Authorization", "X-Request-Id")

	tests := []struct {
		name   string
		header http.Header
		want   bool
	}{
		{
			name: "all_present",
			header: http.Header{
				"Authorization": {"Bearer x"},
				"X-Request-Id":  {"abc"},
			},
			want: true,
		},
		{
			name:   "missing_auth",
			header: http.Header{"X-Request-Id": {"abc"}},
			want:   false,
		},
		{
			name:   "empty_value",
			header: http.Header{"Authorization": {}, "X-Request-Id": {"abc"}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeReq("GET", "/")
			req.Header = tt.header
			if got := f.Match(req, nil); got != tt.want {
				t.Errorf("HeaderFilter = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllFilters(t *testing.T) {
	f := AllFilters(
		MethodFilter("POST"),
		URLFilter("/api/"),
	)

	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{name: "both_match", method: "POST", path: "/api/x", want: true},
		{name: "method_only", method: "POST", path: "/other", want: false},
		{name: "path_only", method: "GET", path: "/api/x", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeReq(tt.method, tt.path)
			if got := f.Match(req, nil); got != tt.want {
				t.Errorf("AllFilters = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnyFilter(t *testing.T) {
	f := AnyFilter(MethodFilter("POST"), MethodFilter("PUT"))

	tests := []struct {
		name   string
		method string
		want   bool
	}{
		{name: "post", method: "POST", want: true},
		{name: "put", method: "PUT", want: true},
		{name: "get", method: "GET", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeReq(tt.method, "/")
			if got := f.Match(req, nil); got != tt.want {
				t.Errorf("AnyFilter = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFilter(t *testing.T) {
	f := NotFilter(URLFilter("/healthz"))

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "not_health", path: "/api", want: true},
		{name: "is_health", path: "/healthz", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := makeReq("GET", tt.path)
			if got := f.Match(req, nil); got != tt.want {
				t.Errorf("NotFilter = %v, want %v", got, tt.want)
			}
		})
	}
}
