package coginit

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitForSocket_Found(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	// Create socket file
	f, err := os.Create(sockPath)
	if err != nil {
		t.Fatalf("failed to create socket file: %v", err)
	}
	_ = f.Close()

	// Should find it immediately
	waitForSocket(sockPath, 500*time.Millisecond)
}

func TestWaitForSocket_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "nonexistent.sock")

	// Should timeout (logs warning, doesn't error)
	start := time.Now()
	waitForSocket(sockPath, 300*time.Millisecond)
	elapsed := time.Since(start)

	if elapsed < 200*time.Millisecond {
		t.Fatalf("returned too quickly: %v", elapsed)
	}
}

func TestWaitForSocket_EventuallyAppears(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "late.sock")

	// Create socket after a delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		f, _ := os.Create(sockPath)
		_ = f.Close()
	}()

	waitForSocket(sockPath, 2*time.Second)
}

func TestWaitForHTTP_Found(t *testing.T) {
	// Start a test HTTP server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() { _ = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})) }()

	waitForHTTP("http://"+ln.Addr().String()+"/health", 2*time.Second)
}

func TestWaitForHTTP_Timeout(t *testing.T) {
	// Connect to a port that's not listening
	waitForHTTP("http://127.0.0.1:19999/health", 300*time.Millisecond)
	// Should just log warning, not error
}
