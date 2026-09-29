package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/delaakakpo/tunnel-inspector/internal/config"
	"github.com/delaakakpo/tunnel-inspector/internal/requests"
	"github.com/delaakakpo/tunnel-inspector/internal/server"
	"github.com/delaakakpo/tunnel-inspector/internal/tunnel"
	"github.com/delaakakpo/tunnel-inspector/internal/websocket"
)

func TestServer_API(t *testing.T) {
	// Upstream mock server
	var upstreamReceivedPath string
	var upstreamReceivedBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamReceivedPath = r.URL.Path
		upstreamReceivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"replayed":true}`))
	}))
	defer upstream.Close()

	// Parse upstream port
	cfg := config.NewDefaultConfig()
	// Set target to upstream server
	cfg.AppHost = "127.0.0.1"
	// Parse port from upstream.URL
	var upstreamPort int
	_, _ = io.WriteString(io.Discard, upstream.URL)
	// We can set AppPort from upstream listener
	upstreamAddr := upstream.Listener.Addr().String()
	_, portStr, _ := netSplitHostPort(upstreamAddr)
	upstreamPort = mustAtoi(portStr)
	cfg.AppPort = upstreamPort

	dbPath := filepath.Join(t.TempDir(), "server_test.db")
	repo, err := requests.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}
	defer repo.Close()

	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	tun := tunnel.NewMockTunnel("https://test-tunnel.trycloudflare.com")
	srv := server.NewServer(cfg, repo, hub, tun, nil)
	handler := srv.Handler()

	// 1. Seed a transaction in repo
	origTx := &requests.HTTPTransaction{
		ID:        "req_initial",
		Timestamp: time.Now().UTC(),
		Duration:  30,
		Request: requests.RequestData{
			Method:  "POST",
			URL:     "/webhook",
			Path:    "/webhook",
			Headers: map[string][]string{"Content-Type": {"application/json"}},
			Body:    []byte(`{"msg":"original"}`),
			Size:    18,
		},
		Response: requests.ResponseData{
			StatusCode: 200,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       []byte(`{"ok":true}`),
			Size:       11,
		},
	}
	_ = repo.Save(t.Context(), origTx)

	// 2. Test GET /api/tunnel
	req := httptest.NewRequest("GET", "/api/tunnel", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/tunnel, got %d", rec.Code)
	}
	var tunnelResp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&tunnelResp)
	if tunnelResp["url"] != "https://test-tunnel.trycloudflare.com" {
		t.Errorf("unexpected tunnel url: %v", tunnelResp["url"])
	}

	// 3. Test GET /api/status
	req = httptest.NewRequest("GET", "/api/status", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/status, got %d", rec.Code)
	}
	var statusResp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&statusResp)
	if statusResp["status"] != "ok" {
		t.Errorf("unexpected status: %v", statusResp["status"])
	}

	// 4. Test GET /api/requests
	req = httptest.NewRequest("GET", "/api/requests", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/requests, got %d", rec.Code)
	}
	var listResp struct {
		Requests []requests.RequestSummary `json:"requests"`
		Total    int                       `json:"total"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&listResp)
	if listResp.Total != 1 || len(listResp.Requests) != 1 {
		t.Errorf("expected 1 request, got total=%d len=%d", listResp.Total, len(listResp.Requests))
	}

	// 5. Test GET /api/requests/:id
	req = httptest.NewRequest("GET", "/api/requests/req_initial", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/requests/req_initial, got %d", rec.Code)
	}
	var itemResp requests.HTTPTransaction
	_ = json.NewDecoder(rec.Body).Decode(&itemResp)
	if itemResp.ID != "req_initial" {
		t.Errorf("expected req_initial, got %s", itemResp.ID)
	}

	// 6. Test POST /api/requests/:id/replay
	replayBody := `{"body":"{\"msg\":\"modified\"}"}`
	req = httptest.NewRequest("POST", "/api/requests/req_initial/replay", bytes.NewReader([]byte(replayBody)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for replay, got %d: %s", rec.Code, rec.Body.String())
	}
	if upstreamReceivedPath != "/webhook" {
		t.Errorf("expected upstream path /webhook, got %s", upstreamReceivedPath)
	}
	if string(upstreamReceivedBody) != `{"msg":"modified"}` {
		t.Errorf("expected upstream body {\"msg\":\"modified\"}, got %s", string(upstreamReceivedBody))
	}

	// Verify we now have 2 requests in repo
	listAfterReplay, countAfterReplay, _ := repo.List(t.Context(), requests.RequestFilter{})
	if countAfterReplay != 2 || len(listAfterReplay) != 2 {
		t.Errorf("expected 2 requests after replay, got %d", countAfterReplay)
	}

	// 7. Test DELETE /api/requests
	req = httptest.NewRequest("DELETE", "/api/requests", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for clear, got %d", rec.Code)
	}
	_, countAfterClear, _ := repo.List(t.Context(), requests.RequestFilter{})
	if countAfterClear != 0 {
		t.Errorf("expected 0 requests after clear, got %d", countAfterClear)
	}
}

func netSplitHostPort(addr string) (string, string, error) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
	}
	return addr, "", nil
}

func mustAtoi(s string) int {
	var res int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			res = res*10 + int(c-'0')
		}
	}
	return res
}
