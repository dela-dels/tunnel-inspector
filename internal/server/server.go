package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dela-dels/tunnel-inspector/internal/config"
	"github.com/dela-dels/tunnel-inspector/internal/proxy"
	"github.com/dela-dels/tunnel-inspector/internal/requests"
	"github.com/dela-dels/tunnel-inspector/internal/tunnel"
	"github.com/dela-dels/tunnel-inspector/internal/websocket"
)

// Server coordinates the Dashboard HTTP server, API endpoints, and WebSocket hub.
type Server struct {
	cfg        *config.Config
	repo       requests.Repository
	hub        *websocket.Hub
	tun        tunnel.Tunnel
	httpServer *http.Server
	staticFS   fs.FS
}

// NewServer creates a new Server instance.
func NewServer(
	cfg *config.Config,
	repo requests.Repository,
	hub *websocket.Hub,
	tun tunnel.Tunnel,
	staticFS fs.FS,
) *Server {
	return &Server{
		cfg:      cfg,
		repo:     repo,
		hub:      hub,
		tun:      tun,
		staticFS: staticFS,
	}
}

// Handler returns the root HTTP handler for the dashboard and API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.HandleWebSocket(s.hub, w, r)
	})

	// API routes
	mux.HandleFunc("/api/tunnel", s.handleTunnel)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/requests", s.handleRequests)
	mux.HandleFunc("/api/requests/", s.handleRequestItem)

	// Static SPA files or fallback
	var fileHandler http.Handler
	if s.staticFS != nil {
		fileHandler = spaHandler(s.staticFS)
	} else {
		fileHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Tunnel Inspector</title></head>
<body style="font-family: sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem;">
<h2>Tunnel Inspector API Server</h2>
<p>Dashboard UI is running in development mode on <a href="http://localhost:5173" style="color:#38bdf8">http://localhost:5173</a> or compile the web assets.</p>
</body>
</html>`))
		})
	}

	return corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws" {
			mux.ServeHTTP(w, r)
			return
		}
		fileHandler.ServeHTTP(w, r)
	}))
}

// Start begins listening on the configured dashboard host and port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.DashboardHost, s.cfg.DashboardPort)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	urlStr := ""
	var status tunnel.Status = tunnel.StatusStopped
	if s.tun != nil {
		urlStr = s.tun.URL()
		status = s.tun.Status()
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"url":    urlStr,
		"status": status,
		"target": s.cfg.UpstreamURL(),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	urlStr := ""
	var status tunnel.Status = tunnel.StatusStopped
	if s.tun != nil {
		urlStr = s.tun.URL()
		status = s.tun.Status()
	}

	total := 0
	if s.repo != nil {
		_, count, err := s.repo.List(r.Context(), requests.RequestFilter{Limit: 1})
		if err == nil {
			total = count
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"app_target":    s.cfg.UpstreamURL(),
		"tunnel_url":    urlStr,
		"tunnel_status": status,
		"dashboard_url": s.cfg.DashboardURL(),
		"proxy_url":     s.cfg.ProxyURL(),
		"request_count": total,
	})
}

func (s *Server) handleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		status, _ := strconv.Atoi(r.URL.Query().Get("status"))
		method := r.URL.Query().Get("method")
		search := r.URL.Query().Get("search")

		filter := requests.RequestFilter{
			Page:   page,
			Limit:  limit,
			Status: status,
			Method: method,
			Search: search,
		}

		summaries, total, err := s.repo.List(r.Context(), filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if page < 1 {
			page = 1
		}
		if limit < 1 {
			limit = 50
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"requests": summaries,
			"total":    total,
			"page":     page,
			"limit":    limit,
		})

	case http.MethodDelete:
		if err := s.repo.Clear(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.hub != nil {
			_ = s.hub.BroadcastJSON(map[string]any{
				"type": "requests.cleared",
			})
		}
		respondJSON(w, http.StatusOK, map[string]any{"cleared": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRequestItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/requests/")
	if path == "" {
		s.handleRequests(w, r)
		return
	}

	parts := strings.Split(path, "/")
	id := parts[0]

	// Check if this is replay: /api/requests/:id/replay
	if len(parts) > 1 && parts[1] == "replay" {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleReplay(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tx, err := s.repo.GetByID(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tx == nil {
			http.Error(w, "Request not found", http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, tx)

	case http.MethodDelete:
		if err := s.repo.DeleteByID(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if s.hub != nil {
			_ = s.hub.BroadcastJSON(map[string]any{
				"type": "request.deleted",
				"id":   id,
			})
		}
		respondJSON(w, http.StatusOK, map[string]any{"deleted": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type replayPayload struct {
	Body    *string `json:"body,omitempty"`
	Method  *string `json:"method,omitempty"`
	Path    *string `json:"path,omitempty"`
	Headers *map[string][]string `json:"headers,omitempty"`
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tx == nil {
		http.Error(w, "Transaction not found for replay", http.StatusNotFound)
		return
	}

	var payload replayPayload
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	method := tx.Request.Method
	if payload.Method != nil && *payload.Method != "" {
		method = *payload.Method
	}

	targetPath := tx.Request.Path
	if payload.Path != nil && *payload.Path != "" {
		targetPath = *payload.Path
	}

	bodyBytes := tx.Request.Body
	if payload.Body != nil {
		bodyBytes = []byte(*payload.Body)
	}

	// Build target URL
	targetURL := fmt.Sprintf("%s%s", s.cfg.UpstreamURL(), targetPath)
	if len(tx.Request.Query) > 0 {
		values := url.Values(tx.Request.Query)
		targetURL = fmt.Sprintf("%s?%s", targetURL, values.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to construct replay request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers
	headersToUse := tx.Request.Headers
	if payload.Headers != nil {
		headersToUse = *payload.Headers
	}
	for k, vals := range headersToUse {
		if strings.EqualFold(k, "content-length") {
			continue
		}
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("X-Tunnel-Inspector-Replay", "true")

	// Execute request to upstream
	startTime := time.Now()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, reqErr := client.Do(req)

	duration := time.Since(startTime)
	newID := proxy.GenerateID()

	var (
		statusCode  = http.StatusBadGateway
		respHeaders = make(http.Header)
		respBody    []byte
		respSize    int64
	)

	if reqErr != nil {
		errMsg := fmt.Sprintf("502 Bad Gateway: Failed to replay request to %s: %v", targetURL, reqErr)
		respBody = []byte(errMsg)
		respSize = int64(len(respBody))
	} else {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		respHeaders = resp.Header

		captured, _, totalSize, _ := proxy.ReadAndReplaceBody(resp.Body, s.cfg.MaxBodySize)
		respBody = captured
		respSize = totalSize
		if respSize < 0 {
			respSize = int64(len(respBody))
		}
	}

	newTx := proxy.BuildTransaction(
		newID,
		startTime,
		duration,
		req,
		bodyBytes,
		int64(len(bodyBytes)),
		statusCode,
		respHeaders,
		respBody,
		respSize,
		s.cfg.RedactedHeaders,
	)

	// Save new replayed transaction
	_ = s.repo.Save(ctx, newTx)

	// Broadcast WebSocket event
	if s.hub != nil {
		_ = s.hub.BroadcastJSON(map[string]any{
			"type":    "request.completed",
			"request": newTx.ToSummary(),
		})
	}

	respondJSON(w, http.StatusCreated, newTx)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// spaHandler serves static assets with fallback to index.html for client-side routing.
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		// Check if file exists in filesystem
		f, err := staticFS.Open(cleanPath)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA routes
		indexFile, err := staticFS.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, indexFile)
	})
}
