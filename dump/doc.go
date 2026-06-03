// Package dump provides an http.RoundTripper that captures HTTP exchanges
// and forwards them to a DumpWriter for logging or inspection.
//
// Basic usage:
//
//	client := &http.Client{
//	    Transport: dump.New(http.DefaultTransport,
//	        dump.WithWriter(&dump.SlogWriter{Logger: slog.Default()}),
//	        dump.WithOptions(dump.DumpOptions{RequestHeaders: true}),
//	    ),
//	}
//
// Response dumps: when ResponseBody is enabled, always read and close resp.Body
// (e.g. defer resp.Body.Close()). Without Close, no dump entry is written.
//
// Request body capture uses req.GetBody when set; streaming bodies without GetBody
// (io.Pipe, chunked upload) are not captured (Meta.ReqBodySkipped).
//
// Production: configure WithRequestRedactor / WithResponseRedactor with
// DefaultRedactor for authorization, cookie, and API keys.
package dump
