package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxy struct {
		ListenPort string `yaml:"listen_port"`
	} `yaml:"proxy"`

	Servers []struct {
		Name string `yaml:"name"`
		URL  string `yaml:"url"`
	} `yaml:"servers"`

	Routing struct {
		Rules []struct {
			Model  string `yaml:"model"`
			Server string `yaml:"server"`
		} `yaml:"rules"`
		DefaultServer string `yaml:"default_server"`
	} `yaml:"routing"`
}

type ReverseProxy struct {
	config *Config
}

func main() {
	config, err := loadConfig("config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	proxy := &ReverseProxy{
		config: config,
	}

	fmt.Println("Starting proxy server on", config.Proxy.ListenPort)
	if err := http.ListenAndServe(config.Proxy.ListenPort, proxy); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
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
		model = "" // Use default routing
	}

	// Restore the request body for further use
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Determine target based on the routing rules
	target := p.getTarget(model)

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

func (p *ReverseProxy) getTarget(model string) string {
	// Build a map of model to server name
	routingMap := make(map[string]string)
	for _, rule := range p.config.Routing.Rules {
		routingMap[rule.Model] = rule.Server
	}

	// Get server name based on model, or default server
	serverName := routingMap[model]
	if serverName == "" {
		serverName = p.config.Routing.DefaultServer
	}

	// Find the server URL based on server name
	for _, server := range p.config.Servers {
		if server.Name == serverName {
			return server.URL
		}
	}

	// Fallback to default server if not found
	for _, server := range p.config.Servers {
		if server.Name == p.config.Routing.DefaultServer {
			return server.URL
		}
	}

	// Should not reach here if default server is configured correctly
	log.Printf("Warning: Default server not found, using first server")
	return p.config.Servers[0].URL
}

func isStreamingResponse(resp *http.Response) bool {
	return resp.Header.Get("Transfer-Encoding") == "chunked" ||
		resp.Header.Get("Content-Type") == "text/event-stream" ||
		resp.Header.Get("Content-Type") == "application/octet-stream"
}
