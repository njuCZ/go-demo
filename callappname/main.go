package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	targetAppName := os.Getenv("TARGET_APP_NAME")
	url := fmt.Sprintf("http://%s/", targetAppName)
	for i := 0; i < 100; i++ {
		if err := checkURL(url); err != nil {
			fmt.Printf("[%s] ❌ %s is NOT reachable: %v\n", time.Now().Format(time.RFC3339), url, err)
		} else {
			fmt.Printf("[%s] ✅ %s is reachable\n", time.Now().Format(time.RFC3339), url)
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func checkURL(url string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // success
	}

	return fmt.Errorf("http error: status %d", resp.StatusCode)
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start HTTP server.
	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	err = srv.Shutdown(context.Background())
	return
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// Register handlers.
	mux.Handle("/", http.HandlerFunc(printHeader))

	return mux
}

func printHeader(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!\n\n"))
	w.Write([]byte("headers:\n"))
	for k, v := range r.Header {
		w.Write([]byte(k + ": " + v[0] + "\n"))
	}
}
