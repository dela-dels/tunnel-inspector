package requests_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dela-dels/tunnel-inspector/internal/requests"
)

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := requests.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	// 1. Save and Get
	tx1 := &requests.HTTPTransaction{
		ID:        "req_001",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		Duration:  125,
		Request: requests.RequestData{
			Method:  "POST",
			URL:     "/webhooks/stripe?source=app",
			Path:    "/webhooks/stripe",
			Query:   map[string][]string{"source": {"app"}},
			Headers: map[string][]string{"Content-Type": {"application/json"}, "Authorization": {"[REDACTED]"}},
			Body:    []byte(`{"event":"charge.succeeded"}`),
			Size:    29,
		},
		Response: requests.ResponseData{
			StatusCode: 200,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       []byte(`{"status":"ok"}`),
			Size:       15,
		},
	}

	if err := repo.Save(ctx, tx1); err != nil {
		t.Fatalf("failed to save transaction: %v", err)
	}

	fetched, err := repo.GetByID(ctx, "req_001")
	if err != nil {
		t.Fatalf("failed to get transaction: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected transaction, got nil")
	}
	if fetched.ID != "req_001" {
		t.Errorf("expected ID req_001, got %s", fetched.ID)
	}
	if fetched.Request.Method != "POST" {
		t.Errorf("expected POST, got %s", fetched.Request.Method)
	}
	if string(fetched.Request.Body) != `{"event":"charge.succeeded"}` {
		t.Errorf("body mismatch: %s", string(fetched.Request.Body))
	}
	if fetched.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", fetched.Response.StatusCode)
	}

	// 2. Insert more transactions for listing & filtering
	tx2 := &requests.HTTPTransaction{
		ID:        "req_002",
		Timestamp: time.Now().UTC().Add(time.Second).Truncate(time.Millisecond),
		Duration:  45,
		Request: requests.RequestData{
			Method:  "GET",
			URL:     "/api/users",
			Path:    "/api/users",
			Headers: map[string][]string{"Accept": {"application/json"}},
			Size:    0,
		},
		Response: requests.ResponseData{
			StatusCode: 404,
			Headers:    map[string][]string{},
			Body:       []byte(`{"error":"not found"}`),
			Size:       21,
		},
	}
	if err := repo.Save(ctx, tx2); err != nil {
		t.Fatalf("failed to save tx2: %v", err)
	}

	// List all
	list, total, err := repo.List(ctx, requests.RequestFilter{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}

	// Filter by method
	listMethod, totalMethod, err := repo.List(ctx, requests.RequestFilter{Method: "POST"})
	if err != nil {
		t.Fatalf("failed to list by method: %v", err)
	}
	if totalMethod != 1 || listMethod[0].ID != "req_001" {
		t.Errorf("expected 1 POST item, got %d", totalMethod)
	}

	// Filter by status
	listStatus, totalStatus, err := repo.List(ctx, requests.RequestFilter{Status: 404})
	if err != nil {
		t.Fatalf("failed to list by status: %v", err)
	}
	if totalStatus != 1 || listStatus[0].ID != "req_002" {
		t.Errorf("expected 1 404 item, got %d", totalStatus)
	}

	// Filter by search
	listSearch, totalSearch, err := repo.List(ctx, requests.RequestFilter{Search: "stripe"})
	if err != nil {
		t.Fatalf("failed to list by search: %v", err)
	}
	if totalSearch != 1 || listSearch[0].ID != "req_001" {
		t.Errorf("expected 1 search result, got %d", totalSearch)
	}

	// Delete item
	if err := repo.DeleteByID(ctx, "req_001"); err != nil {
		t.Fatalf("failed to delete req_001: %v", err)
	}
	fetchedDeleted, _ := repo.GetByID(ctx, "req_001")
	if fetchedDeleted != nil {
		t.Errorf("expected nil after delete, got %v", fetchedDeleted)
	}

	// Clear all
	if err := repo.Clear(ctx); err != nil {
		t.Fatalf("failed to clear: %v", err)
	}
	_, totalAfterClear, _ := repo.List(ctx, requests.RequestFilter{})
	if totalAfterClear != 0 {
		t.Errorf("expected 0 items after clear, got %d", totalAfterClear)
	}
}
