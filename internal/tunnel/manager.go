package tunnel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

// ErrCloudflaredNotFound is returned when the cloudflared executable is missing.
var ErrCloudflaredNotFound = errors.New("cloudflared was not found.\n\nInstall Cloudflare cloudflared and try again.")

var tryCloudflareRegex = regexp.MustCompile(`https://[a-zA-Z0-9.-]+\.trycloudflare\.com`)

// Status represents the current state of the tunnel.
type Status string

const (
	StatusStarting  Status = "starting"
	StatusConnected Status = "connected"
	StatusStopped   Status = "stopped"
	StatusError     Status = "error"
)

// Tunnel defines the interface for managing an external tunnel process.
type Tunnel interface {
	Start(ctx context.Context, target string) error
	URL() string
	Status() Status
	Stop() error
}

// StatusListener is a callback function for tunnel status updates.
type StatusListener func(status Status, url string)

// CloudflareTunnel implements Tunnel using the official cloudflared binary.
type CloudflareTunnel struct {
	mu        sync.RWMutex
	url       string
	status    Status
	cmd       *exec.Cmd
	cancelFn  context.CancelFunc
	readyCh   chan struct{}
	listeners []StatusListener
	binPath   string
}

// NewCloudflareTunnel creates a new CloudflareTunnel instance.
func NewCloudflareTunnel() *CloudflareTunnel {
	binPath, _ := exec.LookPath("cloudflared")
	return &CloudflareTunnel{
		status:  StatusStopped,
		binPath: binPath,
	}
}

// OnStatus registers a callback for tunnel status changes.
func (t *CloudflareTunnel) OnStatus(l StatusListener) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.listeners = append(t.listeners, l)
}

func (t *CloudflareTunnel) notifyListeners(status Status, url string) {
	t.mu.RLock()
	listeners := make([]StatusListener, len(t.listeners))
	copy(listeners, t.listeners)
	t.mu.RUnlock()

	for _, l := range listeners {
		l(status, url)
	}
}

// Start launches cloudflared targeting the inspection proxy.
func (t *CloudflareTunnel) Start(ctx context.Context, target string) error {
	t.mu.Lock()
	if t.status == StatusConnected || t.status == StatusStarting {
		t.mu.Unlock()
		return fmt.Errorf("tunnel is already running")
	}

	bin := t.binPath
	if bin == "" {
		var err error
		bin, err = exec.LookPath("cloudflared")
		if err != nil {
			t.status = StatusError
			t.mu.Unlock()
			return ErrCloudflaredNotFound
		}
		t.binPath = bin
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	t.cancelFn = cancel
	t.status = StatusStarting
	t.url = ""
	t.readyCh = make(chan struct{})

	// Arguments to expose local target through TryCloudflare quick tunnel
	args := []string{"tunnel", "--url", target, "--no-autoupdate"}
	cmd := exec.CommandContext(cmdCtx, bin, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.status = StatusError
		t.mu.Unlock()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		t.status = StatusError
		t.mu.Unlock()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		t.status = StatusError
		t.mu.Unlock()
		return fmt.Errorf("failed to start cloudflared: %w", err)
	}

	t.cmd = cmd
	t.mu.Unlock()

	t.notifyListeners(StatusStarting, "")

	// Scan both stdout and stderr concurrently for tunnel URL
	var readyOnce sync.Once
	go t.scanLogs(stdoutPipe, &readyOnce)
	go t.scanLogs(stderrPipe, &readyOnce)

	// Monitor process termination
	go t.monitorProcess(cmd)

	// Wait for URL detection or timeout/cancellation
	select {
	case <-t.readyCh:
		return nil
	case <-time.After(30 * time.Second):
		t.mu.Lock()
		t.status = StatusError
		t.mu.Unlock()
		t.notifyListeners(StatusError, "")
		return fmt.Errorf("timed out waiting for cloudflared public URL")
	case <-ctx.Done():
		_ = t.Stop()
		return ctx.Err()
	}
}

func (t *CloudflareTunnel) scanLogs(r io.Reader, readyOnce *sync.Once) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if match := tryCloudflareRegex.FindString(line); match != "" {
			t.mu.Lock()
			isNew := t.url == ""
			t.url = match
			t.status = StatusConnected
			t.mu.Unlock()

			if isNew {
				readyOnce.Do(func() {
					close(t.readyCh)
				})
				t.notifyListeners(StatusConnected, match)
				log.Printf("[cloudflared] Public tunnel URL detected: %s", match)
			}
		}
	}
}

func (t *CloudflareTunnel) monitorProcess(cmd *exec.Cmd) {
	err := cmd.Wait()

	t.mu.Lock()
	prevStatus := t.status
	if t.status != StatusStopped {
		t.status = StatusError
		if err != nil {
			log.Printf("[cloudflared] process exited with error: %v", err)
		} else {
			log.Printf("[cloudflared] process exited unexpectedly")
		}
	}
	t.url = ""
	t.mu.Unlock()

	if prevStatus != StatusStopped {
		t.notifyListeners(StatusStopped, "")
	}
}

// URL returns the public tunnel URL, or empty string if not connected.
func (t *CloudflareTunnel) URL() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.url
}

// Status returns the current status of the tunnel.
func (t *CloudflareTunnel) Status() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// Stop terminates the cloudflared process gracefully.
func (t *CloudflareTunnel) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.status == StatusStopped {
		return nil
	}
	t.status = StatusStopped

	if t.cancelFn != nil {
		t.cancelFn()
	}

	if t.cmd != nil && t.cmd.Process != nil {
		// Attempt graceful interrupt first
		_ = t.cmd.Process.Signal(os.Interrupt)

		done := make(chan struct{})
		go func() {
			_, _ = t.cmd.Process.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = t.cmd.Process.Kill()
		}
	}

	t.url = ""
	return nil
}

// MockTunnel provides an in-memory tunnel implementation for tests and offline mode.
type MockTunnel struct {
	mu     sync.RWMutex
	url    string
	status Status
}

// NewMockTunnel creates a new mock tunnel with an optional pre-set URL.
func NewMockTunnel(url string) *MockTunnel {
	status := StatusStopped
	if url != "" {
		status = StatusConnected
	}
	return &MockTunnel{
		url:    url,
		status: status,
	}
}

func (m *MockTunnel) Start(ctx context.Context, target string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusConnected
	if m.url == "" {
		m.url = "https://mock-tunnel.trycloudflare.com"
	}
	return nil
}

func (m *MockTunnel) URL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.url
}

func (m *MockTunnel) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *MockTunnel) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusStopped
	m.url = ""
	return nil
}
