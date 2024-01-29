package main

import (
	"github.com/huahuayu/etherscan-code-retriever/api"
	"github.com/huahuayu/etherscan-code-retriever/flags"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"time"
)

func main() {
	flags.Init()
	http.HandleFunc("/code/", loggerMiddleware(api.SourceCodeHandler))
	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}

func loggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() // Record the start time

		// Extract the IP address from the request
		ip := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ip = forwardedFor // Use X-Forwarded-For header if present
		}

		// Call the actual handler
		next.ServeHTTP(w, r)

		// Calculate the duration and log the IP, URL, and the time taken
		duration := time.Since(start)
		log.Printf("IP: %s Request: %s Time: %v", ip, r.URL.Path, duration)
	}
}
