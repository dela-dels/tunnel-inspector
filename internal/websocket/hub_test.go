package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorilla "github.com/gorilla/websocket"
	"github.com/delaakakpo/tunnel-inspector/internal/websocket"
)

func TestHub_Broadcast(t *testing.T) {
	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		websocket.HandleWebSocket(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect first client
	ws1, _, err := gorilla.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial ws1: %v", err)
	}
	defer ws1.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)
	if count := hub.ClientCount(); count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	// Broadcast an event
	testEvent := map[string]any{
		"type": "test.ping",
		"data": "hello",
	}
	if err := hub.BroadcastJSON(testEvent); err != nil {
		t.Fatalf("broadcast error: %v", err)
	}

	// Read message on ws1
	_ = ws1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws1.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read from ws1: %v", err)
	}

	var received map[string]any
	if err := json.Unmarshal(msg, &received); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if received["type"] != "test.ping" || received["data"] != "hello" {
		t.Errorf("unexpected message payload: %s", string(msg))
	}
}
