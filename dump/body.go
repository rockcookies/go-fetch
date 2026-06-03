package dump

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"strings"
)

// captureRequestBody reads up to maxBytes from a GetBody copy without draining req.Body.
// When req.GetBody is nil (streaming uploads, io.Pipe), capture is skipped and skipped is true.
func captureRequestBody(req *http.Request, maxBytes int64, skipBinary bool, contentType string) (data []byte, truncated bool, skipped bool) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, false, false
	}
	if skipBinary && !isTextContent(contentType) {
		return nil, false, false
	}
	if req.GetBody == nil {
		return nil, false, true
	}
	copy, err := req.GetBody()
	if err != nil {
		return nil, false, false
	}
	defer copy.Close()

	data, truncated = readUpToMax(copy, maxBytes)
	_, _ = io.Copy(io.Discard, copy)
	return data, truncated, false
}

func readUpToMax(r io.Reader, maxBytes int64) ([]byte, bool) {
	if maxBytes < 0 {
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, false
		}
		return b, false
	}
	lr := io.LimitReader(r, maxBytes+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, false
	}
	if int64(len(b)) > maxBytes {
		return b[:maxBytes], true
	}
	return b, false
}

// teeBody wraps an io.ReadCloser so reads are forwarded while up to maxBytes
// are captured. onDone fires exactly once on Close.
type teeBody struct {
	src         io.ReadCloser
	capBuf      bytes.Buffer
	capLimit    int64
	skipCapture bool
	truncated   bool
	onDone      func(captured []byte, truncated bool)
	done        bool
}

func newTeeBody(
	src io.ReadCloser,
	maxBytes int64,
	skipBinary bool,
	contentType string,
	onDone func([]byte, bool),
) io.ReadCloser {
	skip := skipBinary && !isTextContent(contentType)
	return &teeBody{
		src:         src,
		capLimit:    maxBytes,
		skipCapture: skip,
		onDone:      onDone,
	}
}

func (tb *teeBody) Read(p []byte) (int, error) {
	n, err := tb.src.Read(p)
	if n > 0 && !tb.done && !tb.skipCapture {
		toCapture := p[:n]
		if tb.capLimit < 0 {
			tb.capBuf.Write(toCapture)
		} else {
			remaining := tb.capLimit - int64(tb.capBuf.Len())
			if remaining <= 0 {
				tb.truncated = true
			} else if int64(len(toCapture)) > remaining {
				tb.capBuf.Write(toCapture[:remaining])
				tb.truncated = true
			} else {
				tb.capBuf.Write(toCapture)
			}
		}
	}
	return n, err
}

func (tb *teeBody) Close() error {
	err := tb.src.Close()
	tb.fire()
	return err
}

func (tb *teeBody) fire() {
	if tb.done {
		return
	}
	tb.done = true
	if tb.onDone != nil {
		tb.onDone(tb.capBuf.Bytes(), tb.truncated)
	}
}

func isTextContent(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		mediaType = ct
	}
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	subtype := mediaType
	if i := strings.LastIndex(mediaType, "/"); i >= 0 {
		subtype = mediaType[i+1:]
	}
	for _, hint := range []string{"json", "xml", "yaml", "javascript", "graphql", "form-urlencoded"} {
		if strings.Contains(subtype, hint) {
			return true
		}
	}
	return false
}
