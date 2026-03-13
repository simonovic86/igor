// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package igor

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Raw WASM import for the HTTP hostcall from the igor host module.

//go:wasmimport igor http_request
func httpRequest(
	methodPtr, methodLen,
	urlPtr, urlLen,
	headersPtr, headersLen,
	bodyPtr, bodyLen,
	respPtr, respCap uint32,
) int32

// HTTPRequest performs an HTTP request through the runtime.
// Requires the "http" capability in the agent manifest.
// Returns the HTTP status code, response body, and any error.
// Headers are passed as a map; nil means no custom headers.
func HTTPRequest(method, url string, headers map[string]string, body []byte) (statusCode int, respBody []byte, err error) {
	methodBuf := []byte(method)
	urlBuf := []byte(url)

	// Encode headers as "Key: Value\n" delimited.
	var headersBuf []byte
	if len(headers) > 0 {
		var sb []byte
		for k, v := range headers {
			sb = append(sb, k...)
			sb = append(sb, ": "...)
			sb = append(sb, v...)
			sb = append(sb, '\n')
		}
		headersBuf = sb
	}

	// Initial response buffer: 8KB.
	respBuf := make([]byte, 8192)

	rc := doHTTPRequest(methodBuf, urlBuf, headersBuf, body, respBuf)

	// If response too large, retry with the size hint.
	if rc == -5 && len(respBuf) >= 4 {
		needed := binary.LittleEndian.Uint32(respBuf[:4])
		if needed > 0 && needed <= 4*1024*1024 { // Cap retry at 4MB.
			respBuf = make([]byte, needed+4) // +4 for the length prefix.
			rc = doHTTPRequest(methodBuf, urlBuf, headersBuf, body, respBuf)
		}
	}

	if rc < 0 {
		return 0, nil, fmt.Errorf("http_request failed: code %d", rc)
	}

	// Parse response: [body_len: 4 bytes LE][body: N bytes].
	if len(respBuf) < 4 {
		return int(rc), nil, nil
	}
	bodyLen := binary.LittleEndian.Uint32(respBuf[:4])
	if int(bodyLen)+4 > len(respBuf) {
		return int(rc), nil, fmt.Errorf("http_request: response body truncated")
	}
	return int(rc), respBuf[4 : 4+bodyLen], nil
}

// doHTTPRequest is the low-level call with pre-allocated buffers.
func doHTTPRequest(method, url, headers, body, resp []byte) int32 {
	var methodPtr, urlPtr, headersPtr, bodyPtr, respPtr uint32
	var headersLen, bodyLen uint32

	methodPtr = uint32(uintptr(unsafe.Pointer(&method[0])))
	urlPtr = uint32(uintptr(unsafe.Pointer(&url[0])))

	if len(headers) > 0 {
		headersPtr = uint32(uintptr(unsafe.Pointer(&headers[0])))
		headersLen = uint32(len(headers))
	}
	if len(body) > 0 {
		bodyPtr = uint32(uintptr(unsafe.Pointer(&body[0])))
		bodyLen = uint32(len(body))
	}
	respPtr = uint32(uintptr(unsafe.Pointer(&resp[0])))

	return httpRequest(
		methodPtr, uint32(len(method)),
		urlPtr, uint32(len(url)),
		headersPtr, headersLen,
		bodyPtr, bodyLen,
		respPtr, uint32(len(resp)),
	)
}

// HTTPGet performs an HTTP GET request.
// Convenience wrapper around HTTPRequest.
func HTTPGet(url string) (statusCode int, body []byte, err error) {
	return HTTPRequest("GET", url, nil, nil)
}

// HTTPPost performs an HTTP POST request with the given content type and body.
// Convenience wrapper around HTTPRequest.
func HTTPPost(url, contentType string, body []byte) (statusCode int, respBody []byte, err error) {
	headers := map[string]string{"Content-Type": contentType}
	return HTTPRequest("POST", url, headers, body)
}
