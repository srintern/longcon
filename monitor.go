package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// connWrapper wraps a net.Conn to detect when it's closed
type connWrapper struct {
	net.Conn
	addr   string
	prefix string
}

func (c *connWrapper) Close() error {
	fmt.Printf("\n%s 🔌 CONNECTION CLOSED to %s", c.prefix, c.addr)
	return c.Conn.Close()
}

const (
	// Hard-coded URLs
	targetURL    = "https://aga-degradation.pacnw.xyz/test_50kb.bin" // URL to hit
	hostOverride = "alias-icn1.vercel.com"                           // IP/hostname to connect to instead of DNS resolution

	// Configuration
	requestInterval = 1 * time.Second
	workerOffset    = 147 * time.Millisecond
	requestTimeout  = 10 * time.Second
)

func main() {
	// Parse command line arguments
	numGoroutines := 1
	if len(os.Args) > 1 {
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			numGoroutines = n
		} else {
			fmt.Printf("Invalid number of goroutines: %s. Using default value of 1.\n", os.Args[1])
		}
	}

	fmt.Printf("Starting HTTP monitor for %s using host override %s with %d goroutine(s)\n",
		targetURL, hostOverride, numGoroutines)
	fmt.Println("Press Ctrl+C to stop...")

	// Set up graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Start worker goroutines
	for i := 0; i < numGoroutines; i++ {
		time.Sleep(workerOffset) // Stagger start times
		go func(workerID int) {
			runWorker(workerID, stopChan)
		}(i + 1)
	}

	// Wait for shutdown signal, bail immediately
	<-stopChan
	fmt.Println("\nShutting down...")
}

// runWorker runs a single worker goroutine that makes periodic HTTP requests
func runWorker(workerID int, stopChan <-chan os.Signal) {
	prefix := fmt.Sprintf("[G%d] ", workerID)
	client := createHTTPClient(prefix)

	// Create a ticker for periodic requests
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	fmt.Printf("%s Worker %d started\n", prefix, workerID)

	// Make initial request
	makeRequest(client, prefix)

	// Main loop
	for {
		select {
		case <-ticker.C:
			makeRequest(client, prefix)
		case <-stopChan:
			fmt.Printf("%s Worker %d stopping...\n", prefix, workerID)
			return
		}
	}
}

func createHTTPClient(prefix string) *http.Client {
	// Create a custom transport with host override and connection reuse
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if hostOverride != "" {
				_, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				addr = net.JoinHostPort(hostOverride, port)
			}
			d := &net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second, // Keep-alive for 30 seconds
			}
			conn, err := d.DialContext(ctx, "tcp4", addr)
			if err == nil {
				fmt.Printf("\n%s 🔗 NEW CONNECTION established to %s", prefix, addr)
				// Wrap the connection to detect when it's closed
				return &connWrapper{Conn: conn, addr: addr, prefix: prefix}, nil
			}
			return conn, err
		},
		// Connection pooling and keep-alive settings
		MaxIdleConns:        1, // Limit to 1 idle connection total
		MaxIdleConnsPerHost: 1, // Limit to 1 idle connection per host
		MaxConnsPerHost:     1,
		IdleConnTimeout:     10 * time.Second, // Keep connection alive for 90 seconds
		DisableKeepAlives:   false,            // Enable keep-alives (default, but explicit)
		// Optional: Disable compression to reduce overhead if not needed
		// DisableCompression: true,
	}

	return &http.Client{
		Transport: transport,
	}
}

func makeRequest(client *http.Client, prefix string) {
	startTime := time.Now()
	timestamp := startTime.Format("15:04:05.000")
	fmt.Printf("\n%s [%s] Start", prefix, timestamp)

	resp, err := client.Get(targetURL)
	endTime := time.Now()
	endTimestamp := endTime.Format("15:04:05.000")

	if err != nil {
		fmt.Printf("\n%s [%s] Error: %v", prefix, endTimestamp, err)
		return
	}
	defer resp.Body.Close()

	// Read headers and display status code and x-vercel-id
	vercelID := resp.Header.Get("x-vercel-id")
	if vercelID == "" {
		vercelID = "not found"
	}

	// Drain the response body to allow for connection reuse and count bytes
	bytesRead, _ := io.Copy(io.Discard, resp.Body)
	duration := endTime.Sub(startTime).Round(time.Millisecond)
	fmt.Printf("\n%s [%s] End - Status: %d, Size: %d bytes, Duration: %s, x-vercel-id: %s", prefix, endTimestamp, resp.StatusCode, bytesRead, duration, vercelID)
}
