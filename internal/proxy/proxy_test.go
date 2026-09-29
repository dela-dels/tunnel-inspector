package proxy_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dela-dels/tunnel-inspector/internal/proxy"
	"github.com/dela-dels/tunnel-inspector/internal/requests"
	"github.com/dela-dels/tunnel-inspector/internal/websocket"
)

func TestProxy_ForwardingAndCapture(t *testing.T) {
	// Upstream test server
	var receivedUpstreamBody []byte
	var receivedUpstreamHeader http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUpstreamHeader = r.Header.Clone()
		receivedUpstreamBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Upstream-Header", "served")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":"created"}`))
	}))
	defer upstream.Close()

	// Setup SQLite repo and WebSocket Hub
	dbPath := filepath.Join(t.TempDir(), "proxy_test.db")
	repo, err := requests.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	p, err := proxy.NewProxy(upstream.URL, repo, hub, 1024*1024, []string{"authorization", "cookie"})
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	// Make request through proxy
	reqBody := `{"name":"test-agent"}`
	req := httptest.NewRequest("POST", "/api/items?dryrun=false", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer top-secret-token")
	req.Header.Set("Cookie", "session=secret-cookie")
	req.Header.Set("X-Custom", "safe-value")

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	// 1. Verify client received response faithfully
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}
	if rec.Body.String() != `{"result":"created"}` {
		t.Errorf("expected body %s, got %s", `{"result":"created"}`, rec.Body.String())
	}
	if rec.Header().Get("X-Upstream-Header") != "served" {
		t.Errorf("expected header X-Upstream-Header to be 'served'")
	}

	// 2. Verify upstream received original (unredacted) headers and body
	if string(receivedUpstreamBody) != reqBody {
		t.Errorf("upstream received body mismatch: %s", string(receivedUpstreamBody))
	}
	if receivedUpstreamHeader.Get("Authorization") != "Bearer top-secret-token" {
		t.Errorf("upstream should receive unredacted authorization header, got: %s", receivedUpstreamHeader.Get("Authorization"))
	}
	if receivedUpstreamHeader.Get("Cookie") != "session=secret-cookie" {
		t.Errorf("upstream should receive unredacted cookie header, got: %s", receivedUpstreamHeader.Get("Cookie"))
	}

	// 3. Verify transaction persisted with redactions
	list, total, err := repo.List(req.Context(), requests.RequestFilter{})
	if err != nil {
		t.Fatalf("failed to list transactions: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 stored transaction, got %d", total)
	}
	if list[0].Method != "POST" || list[0].Path != "/api/items" || list[0].ResponseStatus != 201 {
		t.Errorf("stored summary mismatch: %+v", list[0])
	}

	fullTx, err := repo.GetByID(req.Context(), list[0].ID)
	if err != nil {
		t.Fatalf("failed to fetch full transaction: %v", err)
	}
	if fullTx.Request.Headers["Authorization"][0] != "[REDACTED]" {
		t.Errorf("expected redacted Authorization in DB, got: %v", fullTx.Request.Headers["Authorization"])
	}
	if fullTx.Request.Headers["Cookie"][0] != "[REDACTED]" {
		t.Errorf("expected redacted Cookie in DB, got: %v", fullTx.Request.Headers["Cookie"])
	}
	if fullTx.Request.Headers["X-Custom"][0] != "safe-value" {
		t.Errorf("expected X-Custom to be preserved, got: %v", fullTx.Request.Headers["X-Custom"])
	}
	if string(fullTx.Request.Body) != reqBody {
		t.Errorf("stored request body mismatch: %s", string(fullTx.Request.Body))
	}
	if string(fullTx.Response.Body) != `{"result":"created"}` {
		t.Errorf("stored response body mismatch: %s", string(fullTx.Response.Body))
	}
}

func TestProxy_BodyTruncation(t *testing.T) {
	// Upstream echoes received body size
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(string(rune(len(body)))))
		// Return 100 bytes
		_, _ = w.Write(bytes.Repeat([]byte("B"), 100))
	}))
	defer upstream.Close()

	dbPath := filepath.Join(t.TempDir(), "trunc_test.db")
	repo, _ := requests.NewSQLiteRepository(dbPath)
	defer repo.Close()

	// Max capture size 16 bytes
	maxCapture := int64(16)
	p, _ := proxy.NewProxy(upstream.URL, repo, nil, maxCapture, nil)

	// Send 64 bytes in request
	largeReqBody := bytes.Repeat([]byte("A"), 64)
	req := httptest.NewRequest("POST", "/upload", bytes.NewReader(largeReqBody))
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Check stored transaction
	list, _, _ := repo.List(req.Context(), requests.RequestFilter{})
	tx, _ := repo.GetByID(req.Context(), list[0].ID)

	// Upstream received full stream, but captured body is truncated to maxCapture
	if len(tx.Request.Body) != int(maxCapture) {
		t.Errorf("expected captured request body to be truncated to %d, got %d", maxCapture, len(tx.Request.Body))
	}
	if len(tx.Response.Body) != int(maxCapture) {
		t.Errorf("expected captured response body to be truncated to %d, got %d", maxCapture, len(tx.Response.Body))
	}
}

func TestProxy_UpstreamUnavailable_502(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "502_test.db")
	repo, _ := requests.NewSQLiteRepository(dbPath)
	defer repo.Close()

	// Point to closed port
	p, _ := proxy.NewProxy("http://127.0.0.1:54321", repo, nil, 1024, nil)

	req := httptest.NewRequest("GET", "/failing/endpoint", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}

	// Verify failed transaction was recorded
	list, total, _ := repo.List(req.Context(), requests.RequestFilter{})
	if total != 1 {
		t.Fatalf("expected 1 recorded failed transaction, got %d", total)
	}
	if list[0].ResponseStatus != http.StatusBadGateway {
		t.Errorf("expected 502 status in DB, got %d", list[0].ResponseStatus)
	}
}
