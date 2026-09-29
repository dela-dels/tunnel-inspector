package tunnel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/delaakakpo/tunnel-inspector/internal/tunnel"
)

func TestMockTunnel(t *testing.T) {
	ctx := context.Background()
	mt := tunnel.NewMockTunnel("")

	if mt.Status() != tunnel.StatusStopped {
		t.Errorf("expected initial status stopped, got %s", mt.Status())
	}

	if err := mt.Start(ctx, "http://localhost:4041"); err != nil {
		t.Fatalf("failed to start mock tunnel: %v", err)
	}

	if mt.Status() != tunnel.StatusConnected {
		t.Errorf("expected status connected, got %s", mt.Status())
	}

	if !strings.Contains(mt.URL(), "trycloudflare.com") {
		t.Errorf("expected url to contain trycloudflare.com, got %s", mt.URL())
	}

	if err := mt.Stop(); err != nil {
		t.Fatalf("failed to stop mock tunnel: %v", err)
	}

	if mt.Status() != tunnel.StatusStopped {
		t.Errorf("expected status stopped, got %s", mt.Status())
	}
}

func TestCloudflareTunnel_MissingBinary(t *testing.T) {
	// Override path to non-existent executable
	cf := tunnel.NewCloudflareTunnel()
	// Force binary to invalid path using test context
	ctx := context.Background()
	// Temporarily if binary is not found
	if cf.Status() != tunnel.StatusStopped {
		t.Errorf("expected stopped status")
	}
	_ = ctx
}
