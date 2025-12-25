package fetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Response struct and methods
//_______________________________________________________________________

// Response represents an HTTP response.
type Response struct {
	Request     *Request
	Body        io.ReadCloser
	RawResponse *http.Response
	IsRead      bool

	// Err field used to cascade the response middleware error
	// in the chain
	Err error

	bodyBytes  []byte
	size       int64
	receivedAt time.Time
}

// Status returns the HTTP status string.
func (r *Response) Status() string {
	if r.RawResponse == nil {
		return ""
	}
	return r.RawResponse.Status
}

// StatusCode returns the HTTP status code.
func (r *Response) StatusCode() int {
	if r.RawResponse == nil {
		return 0
	}
	return r.RawResponse.StatusCode
}

// Proto returns the HTTP protocol.
func (r *Response) Proto() string {
	if r.RawResponse == nil {
		return ""
	}
	return r.RawResponse.Proto
}

// Result returns the response value as an object.
func (r *Response) Result() any {
	return r.Request.Result
}

// Error returns the error object.
func (r *Response) Error() any {
	return r.Request.Error
}

// Header returns the response headers.
func (r *Response) Header() http.Header {
	if r.RawResponse == nil {
		return http.Header{}
	}
	return r.RawResponse.Header
}

// Cookies returns all response cookies.
func (r *Response) Cookies() []*http.Cookie {
	if r.RawResponse == nil {
		return make([]*http.Cookie, 0)
	}
	return r.RawResponse.Cookies()
}

// String returns the response body as a string.
// NOTE: Returns empty string on auto-unmarshal unless unlimited reads enabled.
func (r *Response) String() string {
	r.readIfRequired()
	return strings.TrimSpace(string(r.bodyBytes))
}

// Bytes returns the response body as a byte slice.
// NOTE: Returns empty slice on auto-unmarshal unless unlimited reads enabled.
func (r *Response) Bytes() []byte {
	r.readIfRequired()
	return r.bodyBytes
}

// Duration returns the HTTP response time duration.
func (r *Response) Duration() time.Duration {
	if r.Request.trace != nil {
		return r.Request.TraceInfo().TotalTime
	}
	return r.receivedAt.Sub(r.Request.Time)
}

// ReceivedAt returns the time when the response was received.
func (r *Response) ReceivedAt() time.Time {
	return r.receivedAt
}

// Size returns the HTTP response size in bytes.
func (r *Response) Size() int64 {
	r.readIfRequired()
	return r.size
}

// IsSuccess returns true if status code is 200-299.
func (r *Response) IsSuccess() bool {
	return r.StatusCode() > 199 && r.StatusCode() < 300
}

// IsError returns true if status code >= 400.
func (r *Response) IsError() bool {
	return r.StatusCode() > 399
}

// RedirectHistory returns redirect history with URL and status code.
func (r *Response) RedirectHistory() []*RedirectInfo {
	if r.RawResponse == nil {
		return nil
	}

	redirects := make([]*RedirectInfo, 0)
	res := r.RawResponse
	for res != nil {
		req := res.Request
		redirects = append(redirects, &RedirectInfo{
			StatusCode: res.StatusCode,
			URL:        req.URL.String(),
		})
		res = req.Response
	}

	return redirects
}

func (r *Response) setReceivedAt() {
	r.receivedAt = time.Now()
	if r.Request.trace != nil {
		r.Request.trace.endTime = r.receivedAt
	}
}

func (r *Response) fmtBodyString(sl int) string {
	if r.Request.DoNotParseResponse {
		return "***** DO NOT PARSE RESPONSE - Enabled *****"
	}

	bl := len(r.bodyBytes)
	if r.IsRead && bl == 0 {
		return "***** RESPONSE BODY IS ALREADY READ - see Response.{Result()/Error()} *****"
	}

	if bl > 0 {
		if bl > sl {
			return fmt.Sprintf("***** RESPONSE TOO LARGE (size - %d) *****", bl)
		}

		ct := r.Header().Get(hdrContentTypeKey)
		ctKey := inferContentTypeMapKey(ct)
		if jsonKey == ctKey {
			out := acquireBuffer()
			defer releaseBuffer(out)
			err := json.Indent(out, r.bodyBytes, "", "   ")
			if err != nil {
				r.Request.log.Errorf("DebugLog: Response.fmtBodyString: %v", err)
				return ""
			}
			return out.String()
		}
		return r.String()
	}

	return "***** NO CONTENT *****"
}

func (r *Response) readIfRequired() {
	if len(r.bodyBytes) == 0 && !r.Request.DoNotParseResponse {
		_ = r.readAll()
	}
}

var ioReadAll = io.ReadAll

// auto-unmarshal didn't happen, so fallback to
// old behavior of reading response as body bytes
func (r *Response) readAll() (err error) {
	if r.Body == nil || r.IsRead {
		return nil
	}

	if _, ok := r.Body.(*copyReadCloser); ok {
		_, err = ioReadAll(r.Body)
	} else {
		r.bodyBytes, err = ioReadAll(r.Body)
		closeq(r.Body)
		r.Body = &nopReadCloser{r: bytes.NewReader(r.bodyBytes), resetOnEOF: true}
	}
	if err == io.ErrUnexpectedEOF {
		// content-encoding scenario's - empty/no response body from server
		err = nil
	}

	r.IsRead = true
	return
}

func (r *Response) wrapLimitReadCloser() {
	r.Body = &limitReadCloser{
		r: r.Body,
		l: r.Request.ResponseBodyLimit,
		f: func(s int64) {
			r.size = s
		},
	}
}

func (r *Response) wrapCopyReadCloser() {
	r.Body = &copyReadCloser{
		s: r.Body,
		t: acquireBuffer(),
		f: func(b *bytes.Buffer) {
			r.bodyBytes = append([]byte{}, b.Bytes()...)
			closeq(r.Body)
			r.Body = &nopReadCloser{r: bytes.NewReader(r.bodyBytes), resetOnEOF: true}
			releaseBuffer(b)
		},
	}
}

func (r *Response) wrapContentDecompressor() error {
	ce := r.Header().Get(hdrContentEncodingKey)
	if isStringEmpty(ce) {
		return nil
	}

	if decFunc, f := r.Request.client.ContentDecompressors()[ce]; f {
		dec, err := decFunc(r.Body)
		if err != nil {
			if err == io.EOF {
				// empty/no response body from server
				err = nil
			}
			return err
		}

		r.Body = dec
		r.Header().Del(hdrContentEncodingKey)
		r.Header().Del(hdrContentLengthKey)
		r.RawResponse.ContentLength = -1
	} else {
		return ErrContentDecompressorNotFound
	}

	return nil
}
