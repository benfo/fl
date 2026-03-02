package calendar

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// startCallbackServer binds a local HTTP server on a random port, validates
// the state parameter, and returns the redirect URI, code/error channels, and
// a shutdown function. The caller is responsible for calling shutdown().
func startCallbackServer(state string) (redirectURI string, codeCh <-chan string, errCh <-chan error, shutdown func(), err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("binding local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	// Use localhost (not 127.0.0.1) so Microsoft's loopback port-matching works.
	uri := fmt.Sprintf("http://localhost:%d/callback", port)

	codes := make(chan string, 1)
	errs := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if q.Get("state") != state {
			errs <- fmt.Errorf("state mismatch — possible CSRF")
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		if errParam := q.Get("error"); errParam != "" {
			errs <- fmt.Errorf("authorization denied: %s", errParam)
			fmt.Fprintf(w, "<html><body><h2>Authorization denied: %s. You can close this tab.</h2></body></html>", errParam)
			return
		}
		code := q.Get("code")
		if code == "" {
			errs <- fmt.Errorf("no authorization code in callback")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		fmt.Fprintln(w, "<html><body><h2>fl: authorized! You can close this tab.</h2></body></html>")
		codes <- code
	})

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errs <- fmt.Errorf("local server: %w", err)
		}
	}()

	return uri, codes, errs, func() { srv.Close() }, nil
}

// waitForCode blocks until a code arrives on codeCh, an error arrives on
// errCh, or the 2-minute timeout elapses.
func waitForCode(codeCh <-chan string, errCh <-chan error) (string, error) {
	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-time.After(2 * time.Minute):
		return "", fmt.Errorf("timed out waiting for browser authorization (2 min)")
	}
}
