package dump

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
)

// makeReq builds a minimal *http.Request for filter tests.
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

// ─────────────────────────────────────────────────────────────────────────────
// AlwaysFilter
// ─────────────────────────────────────────────────────────────────────────────

func TestAlwaysFilter(t *testing.T) {
	req := makeReq("GET", "/")
	if !AlwaysFilter.Match(req, nil, nil) {
		t.Fatal("AlwaysFilter should always match")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// URLFilter
// ─────────────────────────────────────────────────────────────────────────────

func TestURLFilter(t *testing.T) {
	f := URLFilter("/api/", "/internal/")

	cases := []struct {
		path string
		want bool
	}{
		{"/api/users", true},
		{"/internal/health", true},
		{"/public", false},
		{"/", false},
	}
	for _, c := range cases {
		req := makeReq("GET", c.path)
		if got := f.Match(req, nil, nil); got != c.want {
			t.Errorf("URLFilter(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MethodFilter
// ─────────────────────────────────────────────────────────────────────────────

func TestMethodFilter(t *testing.T) {
	// Factory args are normalised to uppercase; req.Method is always uppercase
	// in net/http, so no normalisation is needed on the match side.
	f := MethodFilter("POST", "put") // "put" in factory arg → stored as "PUT"

	cases := []struct {
		method string
		want   bool
	}{
		{"POST", true},
		{"PUT", true},
		{"GET", false},
		{"DELETE", false},
	}
	for _, c := range cases {
		req := makeReq(c.method, "/")
		if got := f.Match(req, nil, nil); got != c.want {
			t.Errorf("MethodFilter(%q) = %v, want %v", c.method, got, c.want)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// StatusFilter
// ─────────────────────────────────────────────────────────────────────────────

func TestStatusFilter(t *testing.T) {
	f := StatusFilter([2]int{400, 499}, [2]int{500, 599})

	cases := []struct {
		resp *http.Response
		want bool
	}{
		{makeResp(200), false},
		{makeResp(400), true},
		{makeResp(404), true},
		{makeResp(499), true},
		{makeResp(500), true},
		{makeResp(503), true},
		{nil, false}, // nil resp → never matches
	}
	for _, c := range cases {
		req := makeReq("GET", "/")
		if got := f.Match(req, c.resp, nil); got != c.want {
			status := 0
			if c.resp != nil {
				status = c.resp.StatusCode
			}
			t.Errorf("StatusFilter(status=%d) = %v, want %v", status, got, c.want)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HeaderFilter
// ─────────────────────────────────────────────────────────────────────────────

func TestHeaderFilter(t *testing.T) {
	f := HeaderFilter("X-Request-ID", "Authorization")

	withBoth := makeReq("GET", "/")
	withBoth.Header.Set("X-Request-ID", "abc")
	withBoth.Header.Set("Authorization", "Bearer token")

	withOne := makeReq("GET", "/")
	withOne.Header.Set("X-Request-ID", "abc")

	withNone := makeReq("GET", "/")

	cases := []struct {
		req  *http.Request
		want bool
	}{
		{withBoth, true},
		{withOne, false}, // missing Authorization
		{withNone, false},
	}
	for i, c := range cases {
		if got := f.Match(c.req, nil, nil); got != c.want {
			t.Errorf("case %d: HeaderFilter = %v, want %v", i, got, c.want)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ErrorFilter
// ─────────────────────────────────────────────────────────────────────────────

func TestErrorFilter(t *testing.T) {
	f := ErrorFilter()
	req := makeReq("GET", "/")

	if f.Match(req, nil, nil) {
		t.Error("ErrorFilter should not match when err is nil")
	}
	if !f.Match(req, nil, errors.New("dial: connection refused")) {
		t.Error("ErrorFilter should match when err is non-nil")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AllFilters / AnyFilter / NotFilter
// ─────────────────────────────────────────────────────────────────────────────

func TestAllFilters(t *testing.T) {
	always := AlwaysFilter
	never := NotFilter(AlwaysFilter)

	req := makeReq("GET", "/")
	if !AllFilters(always, always).Match(req, nil, nil) {
		t.Error("AllFilters(always,always) should match")
	}
	if AllFilters(always, never).Match(req, nil, nil) {
		t.Error("AllFilters(always,never) should not match")
	}
	if AllFilters().Match(req, nil, nil) != true {
		t.Error("AllFilters() (empty) should vacuously match")
	}
}

func TestAnyFilter(t *testing.T) {
	always := AlwaysFilter
	never := NotFilter(AlwaysFilter)

	req := makeReq("GET", "/")
	if !AnyFilter(never, always).Match(req, nil, nil) {
		t.Error("AnyFilter(never,always) should match")
	}
	if AnyFilter(never, never).Match(req, nil, nil) {
		t.Error("AnyFilter(never,never) should not match")
	}
	if AnyFilter().Match(req, nil, nil) {
		t.Error("AnyFilter() (empty) should vacuously not match")
	}
}

func TestNotFilter(t *testing.T) {
	req := makeReq("GET", "/")
	if NotFilter(AlwaysFilter).Match(req, nil, nil) {
		t.Error("NotFilter(AlwaysFilter) should not match")
	}
	if !NotFilter(NotFilter(AlwaysFilter)).Match(req, nil, nil) {
		t.Error("double-NotFilter should match")
	}
}
