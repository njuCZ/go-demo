package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/redis/go-redis/v9"
)

func main() {
	host := os.Getenv("REDIS_HOST")
	redisUser := os.Getenv("REDIS_USERNAME") // Object (principal) ID of the managed identity
	if host == "" {
		log.Fatal("REDIS_HOST environment variable must be set")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to create credential: %v", err)
	}

	newRedisOptions := func() *redis.Options {
		return &redis.Options{
			Addr:      host + ":6380",
			TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			CredentialsProviderContext: func(ctx context.Context) (string, string, error) {
				token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
					Scopes: []string{"https://redis.azure.com/.default"},
				})
				if err != nil {
					return "", "", err
				}
				return redisUser, token.Token, nil
			},
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rdb := redis.NewClient(newRedisOptions())
		defer func() {
			// sleep 5ms to increase chance of hitting connection limits on server side
			time.Sleep(5 * time.Minute)
			rdb.Close()
		}()
		ctx := context.Background()

		val, err := rdb.Get(ctx, "hello").Result()
		if err == redis.Nil {
			if err := rdb.Set(ctx, "hello", "world", 0).Err(); err != nil {
				http.Error(w, "failed to set key: "+err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintln(w, "Key 'hello' did not exist. Set value to 'world'.")
		} else if err != nil {
			http.Error(w, "failed to get key: "+err.Error(), http.StatusInternalServerError)
			return
		} else {
			fmt.Fprintf(w, "hello: %s\n", val)
		}
	})

	// /stress?n=100 creates N separate Redis clients and pings concurrently
	http.HandleFunc("/stress", func(w http.ResponseWriter, r *http.Request) {
		n := 100
		if v := r.URL.Query().Get("n"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
				n = parsed
			}
		}

		var (
			mu        sync.Mutex
			successes int
			failures  int
			wg        sync.WaitGroup
		)

		log.Printf("Stress test: opening %d connections...\n", n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				client := redis.NewClient(newRedisOptions())
				defer client.Close()

				if err := client.Ping(context.Background()).Err(); err != nil {
					mu.Lock()
					failures++
					mu.Unlock()
					log.Printf("Connection failed: %v", err)
				} else {
					mu.Lock()
					successes++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		msg := fmt.Sprintf("Stress test done: %d connections attempted, %d succeeded, %d failed\n", n, successes, failures)
		log.Print(msg)
		fmt.Fprint(w, msg)
	})

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
