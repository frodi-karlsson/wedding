package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// startHealthzServer starts a real listener on a free port serving the given
// status at /healthz, and points the PORT env var at it. It returns the port.
func startHealthzServer(t *testing.T, status int) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	})
	srv := &httptest.Server{Listener: ln, Config: &http.Server{Handler: mux}}
	srv.Start()
	t.Cleanup(srv.Close)

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	t.Setenv("PORT", port)
	return port
}

func TestRunHealthcheck_Returns0When200(t *testing.T) {
	startHealthzServer(t, http.StatusOK)
	if code := runHealthcheck(); code != 0 {
		t.Errorf("runHealthcheck() = %d, want 0 for a 200 response", code)
	}
}

func TestRunHealthcheck_Returns1WhenNon200(t *testing.T) {
	startHealthzServer(t, http.StatusServiceUnavailable)
	if code := runHealthcheck(); code != 1 {
		t.Errorf("runHealthcheck() = %d, want 1 for a 503 response", code)
	}
}

func TestRunHealthcheck_Returns1WhenUnreachable(t *testing.T) {
	// Point at a port with nothing listening.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close() // free the port so the connection is refused
	t.Setenv("PORT", port)

	if code := runHealthcheck(); code != 1 {
		t.Errorf("runHealthcheck() = %d, want 1 when the server is unreachable", code)
	}
}
