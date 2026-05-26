package fetch

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"sort"
	"strings"
	"time"
)

// MultipartField represents a single field in a multipart/form-data request.
// It can be either a form value or a file upload with progress tracking.
type MultipartField struct {
	Name                    string
	FileName                string
	ContentType             string
	GetReader               func() (io.ReadCloser, error)
	FileSize                int64
	ExtraContentDisposition map[string]string
	ProgressInterval        time.Duration
	ProgressCallback        MultipartFieldCallbackFunc
	Values                  []string
}

// MultipartFieldProgress tracks upload progress for a multipart field.
type MultipartFieldProgress struct {
	Name     string
	FileName string
	FileSize int64
	Written  int64
}

// MultipartFieldCallbackFunc is called periodically during field upload to report progress.
type MultipartFieldCallbackFunc func(MultipartFieldProgress)

// MultipartOptions configures multipart request creation.
type MultipartOptions struct {
	Boundary string
}

var escapeQuotesReplacer = strings.NewReplacer("\\", "\\\\", `"`, `\"`)

func escapeQuotes(s string) string {
	return escapeQuotesReplacer.Replace(s)
}

// isValidMIMEToken reports whether s is a valid RFC 7230 token.
func isValidMIMEToken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= 0x20 || c >= 0x7F {
			return false
		}
		switch c {
		case '(', ')', '<', '>', '@', ',', ';', ':', '\\', '"', '/', '[', ']', '?', '=', '{', '}':
			return false
		}
	}
	return true
}

func createMultipartHeader(mf *MultipartField, contentType string) (textproto.MIMEHeader, error) {
	if strings.ContainsAny(mf.Name, "\r\n") || strings.ContainsAny(mf.FileName, "\r\n") {
		return nil, fmt.Errorf("fetch: Name or FileName contains invalid characters")
	}

	h := make(textproto.MIMEHeader)

	var cd strings.Builder
	fmt.Fprintf(&cd, `form-data; name="%s"`, escapeQuotes(mf.Name))
	if mf.FileName != "" {
		fmt.Fprintf(&cd, `; filename="%s"`, escapeQuotes(mf.FileName))
	}
	keys := make([]string, 0, len(mf.ExtraContentDisposition))
	for k := range mf.ExtraContentDisposition {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !isValidMIMEToken(k) {
			return nil, fmt.Errorf("fetch: invalid Content-Disposition parameter key %q", k)
		}
		fmt.Fprintf(&cd, `; %s="%s"`, k, escapeQuotes(mf.ExtraContentDisposition[k]))
	}
	h.Set("Content-Disposition", cd.String())

	if contentType != "" {
		h.Set("Content-Type", contentType)
	}

	return h, nil
}

func createMultipart(w *multipart.Writer, mf *MultipartField) error {
	if len(mf.Values) > 0 {
		for _, v := range mf.Values {
			if err := w.WriteField(mf.Name, v); err != nil {
				return err
			}
		}

		return nil
	}

	content, err := mf.GetReader()
	if err != nil {
		return err
	}
	defer content.Close()

	buf := make([]byte, 512)
	var seeEOF bool
	size, err := content.Read(buf)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return err
		}
		seeEOF = true
	}

	contentType := mf.ContentType
	if contentType == "" && size > 0 {
		contentType = http.DetectContentType(buf[:size])
	}

	header, err := createMultipartHeader(mf, contentType)
	if err != nil {
		return err
	}

	pw, err := w.CreatePart(header)
	if err != nil {
		return err
	}

	if mf.ProgressCallback != nil {
		interval := mf.ProgressInterval

		if interval <= 0 {
			interval = 1 * time.Second
		}

		pw = &callbackWriter{
			Writer:    pw,
			lastTime:  time.Now(),
			interval:  interval,
			totalSize: mf.FileSize,
			callback: func(written int64) {
				mf.ProgressCallback(MultipartFieldProgress{
					Name:     mf.Name,
					FileName: mf.FileName,
					FileSize: mf.FileSize,
					Written:  written,
				})
			},
		}
	}

	if _, err = pw.Write(buf[:size]); err != nil {
		return err
	}

	if seeEOF {
		return nil
	}

	_, err = io.Copy(pw, content)
	return err
}

// Multipart creates middleware that builds a multipart/form-data request body.
// It streams the fields using a pipe to avoid loading everything into memory.
// Supports progress callbacks for individual fields.
func Multipart(fields []*MultipartField, opts ...func(*MultipartOptions)) Middleware {
	options := applyOptions(&MultipartOptions{}, opts...)

	return func(handler Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			if len(fields) == 0 {
				return handler.Handle(client, req)
			}

			pr, pw := io.Pipe()
			req.Body = pr
			req.GetBody = nil
			w := multipart.NewWriter(pw)

			if options.Boundary != "" {
				w.SetBoundary(options.Boundary)
			}

			req.Header.Set("Content-Type", w.FormDataContentType())

			multipartErrChan := make(chan error, 1)

			go func() {
				defer close(multipartErrChan)
				defer pw.Close()
				defer w.Close()

				ctx := req.Context()
				for _, mf := range fields {
					if err := ctx.Err(); err != nil {
						multipartErrChan <- err
						return
					}
					if err := createMultipart(w, mf); err != nil {
						multipartErrChan <- err
						return
					}
				}
			}()

			resp, respErr := handler.Handle(client, req)

			if respErr != nil {
				// Handler failed: close the read end so the writer goroutine
				// unblocks immediately and exits with the handler's error.
				pr.CloseWithError(respErr)
			} else {
				// Handler succeeded but may not have consumed the request body
				// (e.g. it handed off reading to a background goroutine, or it
				// is a test double that never reads at all). Start a drain
				// goroutine so the writer goroutine can finish and propagate any
				// source-read errors through multipartErrChan rather than
				// getting stuck on a blocking pipe write.
				go io.Copy(io.Discard, pr)
			}

			if err := <-multipartErrChan; err != nil && respErr == nil {
				respErr = err
			}

			pr.Close()
			return resp, respErr
		})
	}
}
