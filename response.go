package fetch

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"os"
)

// Response wraps an HTTP response and provides convenient methods
// for reading and decoding the response body.
// It implements io.Reader and buffers content for multiple reads.
type Response struct {
	Error       error
	Header      http.Header
	Cookies     []*http.Cookie
	RawRequest  *http.Request
	RawResponse *http.Response
	buffer      *bytes.Buffer
}

func buildResponse(req *http.Request, resp *http.Response, err error) *Response {
	response := &Response{
		Error:       err,
		Header:      http.Header{},
		Cookies:     []*http.Cookie{},
		RawRequest:  req,
		RawResponse: resp,
		buffer:      bytes.NewBuffer(nil),
	}

	if err != nil {
		return response
	}

	response.Header = resp.Header
	response.Cookies = resp.Cookies()

	return response
}

// Read implements io.Reader by reading from the underlying response body.
// Returns an error if the response contains an error.
func (r *Response) Read(p []byte) (n int, err error) {
	if r.Error != nil {
		return -1, r.Error
	}
	return r.RawResponse.Body.Read(p)
}

// Close discards any remaining response body and closes it.
// Safe to call even when Error is present or RawResponse is nil.
func (r *Response) Close() error {
	if r.Error != nil {
		return r.Error
	}
	if r.RawResponse == nil || r.RawResponse.Body == nil {
		return nil
	}
	io.Copy(io.Discard, r.RawResponse.Body)
	return r.RawResponse.Body.Close()
}

// SaveToFile writes the response body to a file.
// Uses internal buffering if available.
func (r *Response) SaveToFile(fileName string) error {
	if r.Error != nil {
		return r.Error
	}

	fd, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer r.Close() // This is a noop if we use the internal ByteBuffer
	defer fd.Close()

	_, err = io.Copy(fd, r.getInternalReader())
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// JSON decodes the response body as JSON into the provided struct.
func (r *Response) JSON(userStruct any) error {
	if r.Error != nil {
		return r.Error
	}

	jsonDecoder := json.NewDecoder(r.getInternalReader())
	defer r.Close()

	if err := jsonDecoder.Decode(&userStruct); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// XML decodes the response body as XML into the provided struct.
func (r *Response) XML(userStruct any) error {
	if r.Error != nil {
		return r.Error
	}

	xmlDecoder := xml.NewDecoder(r.getInternalReader())
	defer r.Close()

	if err := xmlDecoder.Decode(&userStruct); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// Bytes returns the response body as a byte slice.
// Uses internal buffering for efficient multiple reads.
func (r *Response) Bytes() []byte {
	if r.Error != nil {
		return nil
	}

	r.populateResponseByteBuffer()

	// Are we still empty?
	if r.buffer.Len() == 0 {
		return nil
	}
	return r.buffer.Bytes()
}

// String returns the response body as a string.
// Uses internal buffering for efficient multiple reads.
func (r *Response) String() string {
	if r.Error != nil {
		return ""
	}

	r.populateResponseByteBuffer()
	return r.buffer.String()
}

// ClearInternalBuffer resets the internal buffer.
// Does nothing if an error is present.
func (r *Response) ClearInternalBuffer() {
	if r.Error != nil {
		return // This is a noop as we will be dereferencing a null pointer
	}
	r.buffer.Reset()
}

func (r *Response) populateResponseByteBuffer() {
	// Have I done this already?
	if r.buffer.Len() != 0 {
		return
	}
	defer r.Close()

	// Is there any content?
	if r.RawResponse.ContentLength == 0 {
		return
	}

	// Did the server tell us how big the response is going to be?
	if r.RawResponse.ContentLength > 0 {
		r.buffer.Grow(int(r.RawResponse.ContentLength))
	}

	_, err := io.Copy(r.buffer, r)
	if err != nil && err != io.EOF {
		r.Error = err
		r.RawResponse.Body.Close()
	}
}

func (r *Response) getInternalReader() io.Reader {
	if r.buffer.Len() != 0 {
		return r.buffer
	}
	return r
}
