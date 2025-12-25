// Copyright (c) 2015-present Jeevanandam M (jeeva@myjeeva.com), All rights reserved.
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.
// SPDX-License-Identifier: MIT

// package fetch provides Simple HTTP, REST, and SSE client library for Go.
package fetch // import "resty.dev/v3"

import (
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
)

// Version # of resty
const Version = "3.0.0-beta.5"

// New method creates a new Resty client.
func New() *Client {
	return NewWithTransport(nil)
}

// NewWithTransport method creates a new Resty client with the given [http.RoundTripper].
// If transport is nil, [http.DefaultTransport] is used.
func NewWithTransport(transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return NewWithClient(&http.Client{
		Jar:       createCookieJar(),
		Transport: transport,
	})
}

// NewWithClient method creates a new Resty client with given [http.Client].
func NewWithClient(hc *http.Client) *Client {
	return createClient(hc)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//_______________________________________________________________________

func createCookieJar() *cookiejar.Jar {
	cookieJar, _ := cookiejar.New(nil)
	return cookieJar
}

func createClient(hc *http.Client) *Client {
	c := &Client{ // not setting language default values
		lock:                    &sync.RWMutex{},
		queryParams:             url.Values{},
		formData:                url.Values{},
		header:                  http.Header{},
		cookies:                 make([]*http.Cookie, 0),
		pathParams:              make(map[string]string),
		jsonEscapeHTML:          true,
		httpClient:              hc,
		debugBodyLimit:          math.MaxInt32,
		contentTypeEncoders:     make(map[string]ContentTypeEncoder),
		contentTypeDecoders:     make(map[string]ContentTypeDecoder),
		contentDecompressorKeys: make([]string, 0),
		contentDecompressors:    make(map[string]ContentDecompressor),
	}

	// Logger
	c.SetLogger(createLogger())
	c.SetDebugLogFormatter(DebugLogFormatter)

	c.AddContentTypeEncoder(jsonKey, encodeJSON)
	c.AddContentTypeEncoder(xmlKey, encodeXML)

	c.AddContentTypeDecoder(jsonKey, decodeJSON)
	c.AddContentTypeDecoder(xmlKey, decodeXML)

	// Order matter, giving priority to gzip
	c.AddContentDecompressor("deflate", decompressDeflate)
	c.AddContentDecompressor("gzip", decompressGzip)

	// request middlewares
	c.SetRequestMiddlewares(
		PrepareRequestMiddleware,
	)

	// response middlewares
	c.SetResponseMiddlewares(
		AutoParseResponseMiddleware,
	)

	return c
}
