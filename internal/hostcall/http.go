// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	// Default limits for HTTP hostcall.
	defaultHTTPTimeoutMs    = 10_000
	defaultMaxResponseBytes = 1 << 20 // 1 MB
	maxURLBytes             = 8192
	maxMethodBytes          = 16
	maxHeadersBytes         = 32768
	maxRequestBodyBytes     = 1 << 20 // 1 MB

	// HTTP hostcall error codes (negative i32).
	httpErrNetwork      int32 = -1
	httpErrInputTooLong int32 = -2
	httpErrHostBlocked  int32 = -3
	httpErrTimeout      int32 = -4
	httpErrRespTooLarge int32 = -5
)

// HTTPClient is the interface for executing HTTP requests.
// Defaults to http.DefaultClient; override via SetHTTPClient for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpParams holds the parsed inputs from WASM memory for an HTTP request.
type httpParams struct {
	method  string
	url     string
	headers map[string]string
	body    io.Reader
}

// readHTTPParams reads and validates the HTTP request parameters from WASM memory.
// Returns nil params and an error code on failure.
func readHTTPParams(mem api.Memory,
	methodPtr, methodLen, urlPtr, urlLen,
	headersPtr, headersLen, bodyPtr, bodyLen uint32,
) (*httpParams, int32) {
	if methodLen > maxMethodBytes || urlLen > maxURLBytes ||
		headersLen > maxHeadersBytes || bodyLen > maxRequestBodyBytes {
		return nil, httpErrInputTooLong
	}

	methodData, ok := mem.Read(methodPtr, methodLen)
	if !ok {
		return nil, httpErrNetwork
	}
	urlData, ok := mem.Read(urlPtr, urlLen)
	if !ok {
		return nil, httpErrNetwork
	}

	p := &httpParams{
		method: string(methodData),
		url:    string(urlData),
	}

	if headersLen > 0 {
		headersData, ok := mem.Read(headersPtr, headersLen)
		if !ok {
			return nil, httpErrNetwork
		}
		p.headers = parseHeaders(string(headersData))
	}

	if bodyLen > 0 {
		bodyData, ok := mem.Read(bodyPtr, bodyLen)
		if !ok {
			return nil, httpErrNetwork
		}
		p.body = bytes.NewReader(bodyData)
	}

	return p, 0
}

// writeSizeHint writes the response body length to the first 4 bytes of the
// response buffer so the agent can retry with a larger allocation.
func writeSizeHint(mem api.Memory, respPtr, respCap uint32, size int) {
	if respCap >= 4 {
		sizeBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(sizeBuf, uint32(size))
		mem.Write(respPtr, sizeBuf)
	}
}

// registerHTTP registers the http_request hostcall on the igor WASM host module.
//
// ABI:
//
//	http_request(
//	  method_ptr, method_len,
//	  url_ptr, url_len,
//	  headers_ptr, headers_len,
//	  body_ptr, body_len,
//	  resp_ptr, resp_cap
//	) -> i32
//
// Returns HTTP status code (>0) on success, negative error code on failure.
// Response layout: [body_len: 4 bytes LE][body: N bytes].
func (r *Registry) registerHTTP(builder wazero.HostModuleBuilder, capCfg manifest.CapabilityConfig) {
	client := r.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	allowedHosts := extractAllowedHosts(capCfg)
	timeoutMs := extractIntOption(capCfg.Options, "timeout_ms", defaultHTTPTimeoutMs)
	maxRespBytes := extractIntOption(capCfg.Options, "max_response_bytes", defaultMaxResponseBytes)

	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module,
			methodPtr, methodLen,
			urlPtr, urlLen,
			headersPtr, headersLen,
			bodyPtr, bodyLen,
			respPtr, respCap uint32,
		) int32 {
			params, errCode := readHTTPParams(m.Memory(),
				methodPtr, methodLen, urlPtr, urlLen,
				headersPtr, headersLen, bodyPtr, bodyLen)
			if params == nil {
				return errCode
			}

			if err := checkAllowedHost(params.url, allowedHosts); err != nil {
				r.logger.Warn("HTTP request blocked", "url", params.url, "error", err)
				return httpErrHostBlocked
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			reqCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(reqCtx, params.method, params.url, params.body)
			if err != nil {
				r.logger.Error("HTTP request creation failed", "error", err)
				return httpErrNetwork
			}
			for k, v := range params.headers {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				if reqCtx.Err() != nil {
					return httpErrTimeout
				}
				r.logger.Error("HTTP request failed", "error", err)
				return httpErrNetwork
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxRespBytes)+1))
			if err != nil {
				r.logger.Error("HTTP response read failed", "error", err)
				return httpErrNetwork
			}

			if len(respBody) > maxRespBytes {
				writeSizeHint(m.Memory(), respPtr, respCap, len(respBody))
				return httpErrRespTooLarge
			}

			needed := uint32(4 + len(respBody))
			if needed > respCap {
				writeSizeHint(m.Memory(), respPtr, respCap, len(respBody))
				return httpErrRespTooLarge
			}

			// Write response: [body_len: 4 bytes LE][body: N bytes].
			out := make([]byte, needed)
			binary.LittleEndian.PutUint32(out[:4], uint32(len(respBody)))
			copy(out[4:], respBody)
			if !m.Memory().Write(respPtr, out) {
				return httpErrNetwork
			}

			// Record observation for replay (CM-4).
			obsPayload := make([]byte, 4+len(respBody))
			binary.LittleEndian.PutUint32(obsPayload[:4], uint32(resp.StatusCode))
			copy(obsPayload[4:], respBody)
			r.eventLog.Record(eventlog.HTTPRequest, obsPayload)

			return int32(resp.StatusCode)
		}).
		Export("http_request")
}

// extractAllowedHosts reads the allowed_hosts option from the capability config.
func extractAllowedHosts(cfg manifest.CapabilityConfig) []string {
	if cfg.Options == nil {
		return nil
	}
	raw, ok := cfg.Options["allowed_hosts"]
	if !ok {
		return nil
	}
	slice, ok := raw.([]any)
	if !ok {
		return nil
	}
	hosts := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			hosts = append(hosts, strings.ToLower(s))
		}
	}
	return hosts
}

// extractIntOption reads an integer option with a default fallback.
func extractIntOption(opts map[string]any, key string, defaultVal int) int {
	if opts == nil {
		return defaultVal
	}
	raw, ok := opts[key]
	if !ok {
		return defaultVal
	}
	switch v := raw.(type) {
	case float64:
		return int(v) // JSON numbers decode as float64.
	case int:
		return v
	default:
		return defaultVal
	}
}

// checkAllowedHost validates the request URL against the allowed hosts list.
// If allowedHosts is empty, all hosts are permitted.
func checkAllowedHost(rawURL string, allowedHosts []string) error {
	if len(allowedHosts) == 0 {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	host := strings.ToLower(parsed.Hostname())
	for _, allowed := range allowedHosts {
		if host == allowed {
			return nil
		}
	}
	return fmt.Errorf("host %q not in allowed_hosts", host)
}

// parseHeaders parses "Key: Value\n" delimited headers into a map.
func parseHeaders(raw string) map[string]string {
	headers := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return headers
}
