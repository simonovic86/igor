// SPDX-License-Identifier: Apache-2.0

//go:build !tinygo && !wasip1

package igor

// HTTPRequest performs an HTTP request through the runtime.
// In non-WASM builds, dispatches to the registered MockBackend.
func HTTPRequest(method, url string, headers map[string]string, body []byte) (statusCode int, respBody []byte, err error) {
	if activeMock != nil {
		return activeMock.HTTPRequest(method, url, headers, body)
	}
	panic("igor: HTTPRequest requires WASM runtime or mock (see sdk/igor/mock)")
}

// HTTPGet performs an HTTP GET request.
// In non-WASM builds, dispatches to the registered MockBackend.
func HTTPGet(url string) (statusCode int, body []byte, err error) {
	return HTTPRequest("GET", url, nil, nil)
}

// HTTPPost performs an HTTP POST request with the given content type and body.
// In non-WASM builds, dispatches to the registered MockBackend.
func HTTPPost(url, contentType string, body []byte) (statusCode int, respBody []byte, err error) {
	headers := map[string]string{"Content-Type": contentType}
	return HTTPRequest("POST", url, headers, body)
}
