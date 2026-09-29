package proxy

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dela-dels/tunnel-inspector/internal/requests"
)

// GenerateID produces a unique request identifier (e.g., req_3f8a9b...).
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("req_%s%s", hex.EncodeToString(b), time.Now().Format("150405"))
}

// RedactHeaders creates a copy of headers with sensitive values replaced by [REDACTED].
func RedactHeaders(headers http.Header, redactedKeys []string) map[string][]string {
	result := make(map[string][]string, len(headers))
	redactedMap := make(map[string]bool, len(redactedKeys))
	for _, k := range redactedKeys {
		redactedMap[strings.ToLower(strings.TrimSpace(k))] = true
	}

	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if redactedMap[lowerKey] {
			result[key] = []string{"[REDACTED]"}
		} else {
			cp := make([]string, len(values))
			copy(cp, values)
			result[key] = cp
		}
	}
	return result
}

// CloneQuery creates a copy of url.Values.
func CloneQuery(query map[string][]string) map[string][]string {
	result := make(map[string][]string, len(query))
	for k, v := range query {
		cp := make([]string, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

// ReadAndReplaceBody reads up to maxBytes from the body, and replaces it with
// a new ReadCloser that will supply all remaining bytes to downstream consumers.
// It returns the captured bytes (up to maxBytes), whether truncation occurred, and total bytes if known.
func ReadAndReplaceBody(body io.ReadCloser, maxBytes int64) ([]byte, io.ReadCloser, int64, error) {
	if body == nil || body == http.NoBody {
		return nil, http.NoBody, 0, nil
	}

	// Read up to maxBytes + 1 so we know if it was truncated
	var buf bytes.Buffer
	n, err := io.CopyN(&buf, body, maxBytes+1)
	if err != nil && err != io.EOF {
		_ = body.Close()
		return nil, nil, 0, err
	}

	captured := buf.Bytes()
	var totalRead int64 = n
	var replacement io.ReadCloser

	if n > maxBytes {
		// Truncation: captured more than maxBytes
		capturedBody := make([]byte, maxBytes)
		copy(capturedBody, captured[:maxBytes])

		// Leftover byte from the +1 check
		leftover := captured[maxBytes:]

		// Combined reader for the forwarded stream
		counting := &countingReadCloser{
			reader: io.MultiReader(bytes.NewReader(leftover), body),
			closer: body,
		}
		replacement = io.NopCloser(io.MultiReader(bytes.NewReader(capturedBody), counting))
		return capturedBody, replacement, -1, nil // -1 indicates size unknown until stream completes or ContentLength used
	}

	// Body was <= maxBytes
	_ = body.Close()
	replacement = io.NopCloser(bytes.NewReader(captured))
	return captured, replacement, totalRead, nil
}

type countingReadCloser struct {
	reader io.Reader
	closer io.Closer
	count  int64
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.count += int64(n)
	return n, err
}

func (c *countingReadCloser) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}

// ResponseCapture intercepts and records response status, headers, and body.
type ResponseCapture struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Size       int64
	Duration   time.Duration
}

// BuildTransaction creates an HTTPTransaction from captured request and response data.
func BuildTransaction(
	id string,
	reqTime time.Time,
	duration time.Duration,
	r *http.Request,
	reqBody []byte,
	reqSize int64,
	respStatus int,
	respHeaders http.Header,
	respBody []byte,
	respSize int64,
	redactedHeaders []string,
) *requests.HTTPTransaction {
	if reqSize < 0 {
		reqSize = int64(len(reqBody))
	}
	if respSize < 0 {
		respSize = int64(len(respBody))
	}

	urlStr := r.URL.RequestURI()
	if urlStr == "" {
		urlStr = r.URL.Path
	}

	return &requests.HTTPTransaction{
		ID:        id,
		Timestamp: reqTime,
		Duration:  duration.Milliseconds(),
		Request: requests.RequestData{
			Method:  r.Method,
			URL:     urlStr,
			Path:    r.URL.Path,
			Query:   CloneQuery(r.URL.Query()),
			Headers: RedactHeaders(r.Header, redactedHeaders),
			Body:    reqBody,
			Size:    reqSize,
		},
		Response: requests.ResponseData{
			StatusCode: respStatus,
			Headers:    RedactHeaders(respHeaders, redactedHeaders),
			Body:       respBody,
			Size:       respSize,
		},
		CreatedAt: time.Now().UTC(),
	}
}
