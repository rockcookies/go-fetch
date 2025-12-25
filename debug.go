package fetch

import (
	"fmt"
	"net/http"
	"time"
)

type (
	// DebugLogCallbackFunc is called before debug logging.
	DebugLogCallbackFunc func(*DebugLog)

	// DebugLogFormatterFunc formats debug logs.
	DebugLogFormatterFunc func(*DebugLog) string

	// DebugLog holds request and response debug information.
	DebugLog struct {
		Request   *DebugLogRequest  `json:"request"`
		Response  *DebugLogResponse `json:"response"`
		TraceInfo *TraceInfo        `json:"trace_info"`
	}

	// DebugLogRequest holds request debug information.
	DebugLogRequest struct {
		Host   string      `json:"host"`
		URI    string      `json:"uri"`
		Method string      `json:"method"`
		Proto  string      `json:"proto"`
		Header http.Header `json:"header"`
		Body   string      `json:"body"`
	}

	// DebugLogResponse holds response debug information.
	DebugLogResponse struct {
		StatusCode int           `json:"status_code"`
		Status     string        `json:"status"`
		Proto      string        `json:"proto"`
		ReceivedAt time.Time     `json:"received_at"`
		Duration   time.Duration `json:"duration"`
		Size       int64         `json:"size"`
		Header     http.Header   `json:"header"`
		Body       string        `json:"body"`
	}
)

// DebugLogFormatter formats debug logs in human-readable format.
func DebugLogFormatter(dl *DebugLog) string {
	debugLog := "\n==============================================================================\n"

	req := dl.Request
	debugLog += "~~~ REQUEST ~~~\n" +
		fmt.Sprintf("%s  %s  %s\n", req.Method, req.URI, req.Proto) +
		fmt.Sprintf("HOST   : %s\n", req.Host) +
		fmt.Sprintf("HEADERS:\n%s\n", composeHeaders(req.Header)) +
		fmt.Sprintf("BODY   :\n%v\n", req.Body) +
		"------------------------------------------------------------------------------\n"

	res := dl.Response
	debugLog += "~~~ RESPONSE ~~~\n" +
		fmt.Sprintf("STATUS       : %s\n", res.Status) +
		fmt.Sprintf("PROTO        : %s\n", res.Proto) +
		fmt.Sprintf("RECEIVED AT  : %v\n", res.ReceivedAt.Format(time.RFC3339Nano)) +
		fmt.Sprintf("DURATION     : %v\n", res.Duration) +
		"HEADERS      :\n" +
		composeHeaders(res.Header) + "\n" +
		fmt.Sprintf("BODY         :\n%v\n", res.Body)
	if dl.TraceInfo != nil {
		debugLog += "------------------------------------------------------------------------------\n"
		debugLog += fmt.Sprintf("%v\n", dl.TraceInfo)
	}
	debugLog += "==============================================================================\n"

	return debugLog
}

// DebugLogJSONFormatter function formats the given debug log info in JSON format.
func DebugLogJSONFormatter(dl *DebugLog) string {
	return toJSON(dl)
}

func debugLogger(c *Client, res *Response) {
	req := res.Request
	if !req.Debug {
		return
	}

	rdl := &DebugLogResponse{
		StatusCode: res.StatusCode(),
		Status:     res.Status(),
		Proto:      res.Proto(),
		ReceivedAt: res.ReceivedAt(),
		Duration:   res.Duration(),
		Size:       res.Size(),
		Header:     sanitizeHeaders(res.Header().Clone()),
		Body:       res.fmtBodyString(res.Request.DebugBodyLimit),
	}

	dl := &DebugLog{
		Request:  req.values[debugRequestLogKey].(*DebugLogRequest),
		Response: rdl,
	}

	if res.Request.IsTrace {
		ti := req.TraceInfo()
		dl.TraceInfo = &ti
	}

	dblCallback := c.debugLogCallbackFunc()
	if dblCallback != nil {
		dblCallback(dl)
	}

	formatterFunc := c.debugLogFormatterFunc()
	if formatterFunc != nil {
		debugLog := formatterFunc(dl)
		req.log.Debugf("%s", debugLog)
	}
}

const debugRequestLogKey = "__restyDebugRequestLog"

func prepareRequestDebugInfo(c *Client, r *Request) {
	if !r.Debug {
		return
	}

	rr := r.RawRequest
	rh := rr.Header.Clone()
	if c.Client().Jar != nil {
		for _, cookie := range c.Client().Jar.Cookies(r.RawRequest.URL) {
			s := fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)
			if c := rh.Get(hdrCookieKey); isStringEmpty(c) {
				rh.Set(hdrCookieKey, s)
			} else {
				rh.Set(hdrCookieKey, c+"; "+s)
			}
		}
	}

	rdl := &DebugLogRequest{
		Host:   rr.URL.Host,
		URI:    rr.URL.RequestURI(),
		Method: r.Method,
		Proto:  rr.Proto,
		Header: sanitizeHeaders(rh),
		Body:   r.fmtBodyString(r.DebugBodyLimit),
	}

	r.initValuesMap()
	r.values[debugRequestLogKey] = rdl
}
