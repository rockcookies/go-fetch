package fetch

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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

func createMultipartHeader(mf *MultipartField, contentType string) textproto.MIMEHeader {
	h := make(textproto.MIMEHeader)

	if mf.FileName != "" {
		h.Add("name", mf.Name)
	}

	if mf.FileName != "" {
		h.Add("filename", mf.FileName)
	}

	for k, v := range mf.ExtraContentDisposition {
		h.Add(k, v)
	}

	if contentType != "" {
		h.Set("Content-Type", contentType)
	}

	return h
}

func createMultipart(w *multipart.Writer, mf *MultipartField) error {
	if len(mf.Values) > 0 {
		for _, v := range mf.Values {
			w.WriteField(mf.Name, v)
		}

		return nil
	}

	content, err := mf.GetReader()
	if err != nil {
		return err
	}
	defer content.Close()

	lastTime := time.Now()
	buf := make([]byte, 512)
	seeEOF := false
	size, err := content.Read(buf)
	if err != nil {
		if err == io.EOF {
			seeEOF = true
		} else {
			return err
		}
	}

	contentType := mf.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(buf[:size])
	}

	pw, err := w.CreatePart(createMultipartHeader(mf, contentType))
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
			lastTime:  lastTime,
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

// SetMultipart creates middleware that builds a multipart/form-data request body.
// It streams the fields using a pipe to avoid loading everything into memory.
// Supports progress callbacks for individual fields.
func SetMultipart(fields []*MultipartField, opts ...func(*MultipartOptions)) Middleware {
	options := applyOptions(&MultipartOptions{}, opts...)

	return func(handler Handler) Handler {
		return HandlerFunc(func(client *http.Client, req *http.Request) (*http.Response, error) {
			if len(fields) == 0 {
				return handler.Handle(client, req)
			}

			pr, pw := io.Pipe()
			req.GetBody = func() (io.ReadCloser, error) { return pr, nil }
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

				for _, mf := range fields {
					if err := createMultipart(w, mf); err != nil {
						multipartErrChan <- err
						return
					}
				}
			}()

			resp, respErr := handler.Handle(client, req)
			select {
			case err := <-multipartErrChan:
				respErr = errors.Join(respErr, err)
			default:
				// Channel already consumed or closed, nothing to do
			}

			return resp, respErr
		})
	}
}
