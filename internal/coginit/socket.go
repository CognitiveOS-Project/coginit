package coginit

import (
	"log"
	"net/http"
	"os"
	"time"
)

// waitForSocket polls for a Unix socket file to appear.
func waitForSocket(path string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			log.Printf("[coginit] Socket ready: %s", path)
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	log.Printf("[coginit] Warning: timeout waiting for socket %s", path)
}

// waitForHTTP polls an HTTP endpoint until it responds with 200 OK.
func waitForHTTP(url string, timeout time.Duration) {
	client := http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Printf("[coginit] HTTP ready: %s", url)
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	log.Printf("[coginit] Warning: timeout waiting for HTTP %s", url)
}
