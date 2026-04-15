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

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
)

var accountName = os.Getenv("ACCOUNT_NAME")
var queueName = os.Getenv("QUEUE_NAME")
var queueURL = fmt.Sprintf("https://%s.queue.core.windows.net", accountName)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	msgCount, err := readQueueLength()
	if err != nil {
		fmt.Printf("error read queue length: %+v\n", err)
	} else {
		fmt.Printf("successfully read queue length: %d\n", msgCount)
	}

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
	msgCount, err := readQueueLength()
	if err != nil {
		w.Write([]byte(fmt.Sprintf("error read queue length: %+v\n\n", err)))
	} else {
		w.Write([]byte(fmt.Sprintf("successfully read queue length: %d\n\n", msgCount)))
	}
	w.Write([]byte("headers:\n"))
	for k, v := range r.Header {
		w.Write([]byte(k + ": " + v[0] + "\n"))
	}
}

func readQueueLength() (int32, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		fmt.Printf("failed to create credential: %v", err)
		return 0, err
	}

	// 2. Create Queue client
	client, err := azqueue.NewServiceClient(queueURL, cred, nil)
	if err != nil {
		fmt.Printf("failed to create queue client: %v", err)
		return 0, err
	}

	// 3. Get queue properties (approximate message count)
	resp, err := client.NewQueueClient(queueName).GetProperties(context.Background(), nil)
	if err != nil {
		fmt.Printf("failed to get properties: %v", err)
		return 0, err
	}

	if resp.ApproximateMessagesCount != nil {
		fmt.Printf("Queue length: %d\n", *resp.ApproximateMessagesCount)
		return *resp.ApproximateMessagesCount, nil
	} else {
		fmt.Println("Queue length not available")
		return 0, nil
	}
}
