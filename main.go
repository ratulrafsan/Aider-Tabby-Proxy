package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type ReverseProxy struct {
	server1 string
	server2 string
}

func main() {
	// Create proxy server
	proxy := &ReverseProxy{
		server1: "http://localhost:5001",
		server2: "http://localhost:5002",
	}

	// Start server on port 8080
	fmt.Println("Starting proxy server on :5003")
	if err := http.ListenAndServe(":5003", proxy); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)

	// Limit requests to json types only
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Proxy can only handle incoming JSON contents", http.StatusBadRequest)
		log.Printf("Invalid Content-Type: %s", r.Header.Get("Content-Type"))
		return
	}

	// Read request body to decide which model to pick
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusInternalServerError)
		log.Printf("Error reading request body: %v", err)
		return
	}

	// Extract the "model" property from the request body, assuming it is JSON
	var requestBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		http.Error(w, "Invalid JSON in request body", http.StatusBadRequest)
		log.Printf("Error unmarshalling request body: %v", err)
		return
	}

	model, ok := requestBody["model"].(string)
	if ok {
		log.Printf("Extracted model property: %s", model)
	} else {
		log.Printf("No 'model' property found in the request body")
	}

	// Restore the request body for further use
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	//log.Printf("Request body: %s", string(bodyBytes))

	// Determine target based on URL path
	target := p.targetRoute(model)

	// Parse target URL
	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error parsing target URL: %v", err)
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize proxy director
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		log.Printf("Proxying to: %s%s", target, req.URL.Path)
	}

	// Handle streaming responses
	proxy.ModifyResponse = func(resp *http.Response) error {
		if isStreamingResponse(resp) {
			// Set appropriate headers for streaming
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		}
		return nil
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

// targetRoute determines which server the request should be routed to. Defaults to server1, always
func (p *ReverseProxy) targetRoute(modelName string) string {
	switch modelName {
	case "QwQ-32B":
		return p.server1
	case "Q25_32B-coder-5bpw":
		return p.server1
	}

	return p.server1
}

// isStreamingResponse checks if the response is a streaming response
func isStreamingResponse(resp *http.Response) bool {
	return resp.Header.Get("Transfer-Encoding") == "chunked" ||
		resp.Header.Get("Content-Type") == "text/event-stream" ||
		resp.Header.Get("Content-Type") == "application/octet-stream"
}
