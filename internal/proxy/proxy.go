package proxy

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/delaakakpo/tunnel-inspector/internal/requests"
	"github.com/delaakakpo/tunnel-inspector/internal/websocket"
)

// Proxy handles forwarding requests to the upstream target and capturing metadata.
type Proxy struct {
	targetURL       *url.URL
	reverseProxy    *httputil.ReverseProxy
	repo            requests.Repository
	hub             *websocket.Hub
	maxBodySize     int64
	redactedHeaders []string
}

// NewProxy creates a new inspection proxy.
func NewProxy(
	target string,
	repo requests.Repository,
	hub *websocket.Hub,
	maxBodySize int64,
	redactedHeaders []string,
) (*Proxy, error) {
	parsedURL, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream target %q: %w", target, err)
	}

	p := &Proxy{
		targetURL:       parsedURL,
		repo:            repo,
		hub:             hub,
		maxBodySize:     maxBodySize,
		redactedHeaders: redactedHeaders,
	}

	rp := httputil.NewSingleHostReverseProxy(parsedURL)
	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalHost := req.Host
		originalDirector(req)
		req.Host = parsedURL.Host
		if req.Header.Get("X-Forwarded-Host") == "" {
			req.Header.Set("X-Forwarded-Host", originalHost)
		}
	}

	rp.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("proxy upstream connection error: %v", err)
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(http.StatusBadGateway)
		msg := fmt.Sprintf("502 Bad Gateway: Unable to connect to upstream application at %s. Ensure the application is running.\nError: %v\n", parsedURL.String(), err)
		_, _ = rw.Write([]byte(msg))
	}

	p.reverseProxy = rp
	return p, nil
}

// ServeHTTP inspects, forwards, and records the request/response transaction.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	id := GenerateID()

	// Capture and replace request body safely
	capturedReqBody, newBody, reqSize, err := ReadAndReplaceBody(r.Body, p.maxBodySize)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body = newBody

	if reqSize < 0 && r.ContentLength > 0 {
		reqSize = r.ContentLength
	} else if reqSize < 0 {
		reqSize = int64(len(capturedReqBody))
	}

	// Capture original request headers and query
	reqHeaders := r.Header.Clone()
	if reqHeaders.Get("Host") == "" && r.Host != "" {
		reqHeaders.Set("Host", r.Host)
	}

	// Wrap response writer to capture response status, headers, and body
	recorder := newResponseRecorder(w, p.maxBodySize)

	// Forward through reverse proxy
	p.reverseProxy.ServeHTTP(recorder, r)

	duration := time.Since(startTime)
	statusCode := recorder.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	respSize := recorder.totalSize
	if respSize == 0 && recorder.capturedBody.Len() > 0 {
		respSize = int64(recorder.capturedBody.Len())
	}

	// Build the transaction
	tx := BuildTransaction(
		id,
		startTime,
		duration,
		r,
		capturedReqBody,
		reqSize,
		statusCode,
		recorder.Header(),
		recorder.capturedBody.Bytes(),
		respSize,
		p.redactedHeaders,
	)

	// Persist to repository
	if p.repo != nil {
		if err := p.repo.Save(r.Context(), tx); err != nil {
			log.Printf("failed to persist transaction %s: %v", id, err)
		}
	}

	// Broadcast to WebSocket clients
	if p.hub != nil {
		summary := tx.ToSummary()
		event := map[string]any{
			"type":    "request.completed",
			"request": summary,
		}
		_ = p.hub.BroadcastJSON(event)
	}
}

// responseRecorder captures the response status code, headers, and body while streaming to client.
type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	capturedBody bytes.Buffer
	maxBytes     int64
	totalSize    int64
	wroteHeader  bool
	mu           sync.Mutex
}

func newResponseRecorder(w http.ResponseWriter, maxBytes int64) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		maxBytes:       maxBytes,
	}
}

func (r *responseRecorder) WriteHeader(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.wroteHeader {
		r.statusCode = code
		r.wroteHeader = true
		r.ResponseWriter.WriteHeader(code)
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.wroteHeader {
		r.statusCode = http.StatusOK
		r.wroteHeader = true
	}

	r.totalSize += int64(len(b))

	// Capture up to maxBytes
	remaining := r.maxBytes - int64(r.capturedBody.Len())
	if remaining > 0 {
		if int64(len(b)) <= remaining {
			r.capturedBody.Write(b)
		} else {
			r.capturedBody.Write(b[:remaining])
		}
	}

	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
