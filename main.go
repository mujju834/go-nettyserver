package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var apiGatewayURL, serverPort string

// Load environment variables from .env file
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	log.Println(".env file loaded successfully")

	// Assign environment variables to global variables
	apiGatewayURL = os.Getenv("API_GATEWAY_URL")
	serverPort = os.Getenv("SERVER_PORT")

	fmt.Printf("API_GATEWAY_URL: %s\n", apiGatewayURL)
	fmt.Printf("SERVER_PORT: %s\n", serverPort)
}

// Handler to forward requests to the API Gateway or return a custom message
func proxyHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("Received %s request for %s", req.Method, req.URL.Path)

	// Handle root endpoint with a custom response
	if req.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Netty server deployed by Mujahid in Golang"))
		return
	}

	// Handle CORS preflight requests
	if req.Method == http.MethodOptions {
		setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Create a new request to forward to the API Gateway
	forwardedReq, err := http.NewRequest(req.Method, apiGatewayURL+req.URL.Path, req.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Copy original request headers (excluding CORS-related ones to avoid duplication)
	copyHeaders(req.Header, forwardedReq.Header)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Forward the request to the API Gateway
	resp, err := client.Do(forwardedReq)
	if err != nil {
		log.Printf("Error forwarding request to API Gateway: %v", err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Set the CORS headers
	setCORSHeaders(w)

	// Copy the response headers from the API Gateway, excluding any CORS headers
	copyHeaders(resp.Header, w.Header())

	// Set the appropriate status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body to the client
	io.Copy(w, resp.Body)

	log.Printf("Response from API Gateway: %d", resp.StatusCode)
}

// Helper function to set CORS headers
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
}

// Helper function to copy headers, excluding CORS-related ones
func copyHeaders(from http.Header, to http.Header) {
	for key, values := range from {
		if key == "Access-Control-Allow-Origin" ||
			key == "Access-Control-Allow-Methods" ||
			key == "Access-Control-Allow-Headers" {
			continue // Skip CORS headers to prevent duplication
		}
		for _, value := range values {
			to.Add(key, value)
		}
	}
}

func main() {
	if serverPort == "" {
		log.Fatalf("SERVER_PORT not set or empty")
	}

	http.HandleFunc("/", proxyHandler)

	log.Printf("Starting server on http://localhost:%s", serverPort)
	err := http.ListenAndServe(":"+serverPort, nil)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
