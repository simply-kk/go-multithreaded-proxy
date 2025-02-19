package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// StartServer starts the proxy server
func StartServer() {
	http.HandleFunc("/", handleRequest)
	fmt.Println("Proxy Server is running on port 8080...")
	http.ListenAndServe(":8080", nil)
}

// handleRequest forwards only GET requests to the target server
func handleRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received request for:", r.URL.String())

	// Extract the target URL from the request
	targetURL := strings.TrimPrefix(r.URL.Path, "/")

	// Decode URL (in case of encoded characters)
	targetURL, err := url.QueryUnescape(targetURL)
	if err != nil {
		http.Error(w, "Invalid URL encoding", http.StatusBadRequest)
		return
	}

	// Ensure the URL starts with http:// or https://
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}

	// Forward the GET request
	resp, err := http.Get(targetURL)
	if err != nil {
		http.Error(w, "Failed to reach target server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}
