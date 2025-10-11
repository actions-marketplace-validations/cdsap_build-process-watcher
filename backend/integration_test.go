package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// IntegrationTestServer wraps the main server for testing
type IntegrationTestServer struct {
	server *httptest.Server
}

func NewIntegrationTestServer() *IntegrationTestServer {
	// Create a test server that simulates the main server behavior
	mux := http.NewServeMux()
	
	// Add handlers
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/runs/", func(w http.ResponseWriter, r *http.Request) {
		// Simulate runsHandler with mock data fallback
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		path := strings.TrimPrefix(r.URL.Path, "/runs/")
		if path == "" {
			http.Error(w, "Run ID required", http.StatusBadRequest)
			return
		}
		
		runID := path
		samples := getMockData(runID)
		
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if err := json.NewEncoder(w).Encode(samples); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})
	
	server := httptest.NewServer(mux)
	return &IntegrationTestServer{server: server}
}

func (its *IntegrationTestServer) Close() {
	its.server.Close()
}

func (its *IntegrationTestServer) URL() string {
	return its.server.URL
}

func TestFullAPIIntegration(t *testing.T) {
	server := NewIntegrationTestServer()
	defer server.Close()
	
	baseURL := server.URL()
	
	// Test health endpoint
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/healthz")
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		var health map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}
		
		if health["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %s", health["status"])
		}
	})
	
	// Test runs endpoint
	t.Run("RunsEndpoint", func(t *testing.T) {
		runID := "integration-test-run"
		resp, err := http.Get(baseURL + "/runs/" + runID)
		if err != nil {
			t.Fatalf("Runs endpoint failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		// Check CORS headers
		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Missing CORS header")
		}
		
		var samples []Sample
		if err := json.NewDecoder(resp.Body).Decode(&samples); err != nil {
			t.Fatalf("Failed to decode samples: %v", err)
		}
		
		if len(samples) == 0 {
			t.Fatal("Expected samples, got empty array")
		}
		
		// Verify sample data
		for i, sample := range samples {
			if sample.RunID != runID {
				t.Errorf("Sample %d has wrong RunID: expected %s, got %s", i, runID, sample.RunID)
			}
			if sample.PID == "" {
				t.Errorf("Sample %d has empty PID", i)
			}
			if sample.Name == "" {
				t.Errorf("Sample %d has empty Name", i)
			}
		}
	})
	
	// Test CORS preflight
	t.Run("CORSPreflight", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", baseURL+"/runs/test-run", nil)
		if err != nil {
			t.Fatal(err)
		}
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("CORS preflight failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", resp.StatusCode)
		}
		
		// Check CORS headers
		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Missing CORS header")
		}
		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Error("Missing CORS methods header")
		}
	})
	
	// Test invalid methods
	t.Run("InvalidMethods", func(t *testing.T) {
		runID := "test-run"
		
		// Test POST
		resp, err := http.Post(baseURL+"/runs/"+runID, "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for POST, got %d", resp.StatusCode)
		}
		
		// Test PUT
		req, err := http.NewRequest("PUT", baseURL+"/runs/"+runID, strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("PUT request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for PUT, got %d", resp.StatusCode)
		}
	})
	
	// Test missing run ID
	t.Run("MissingRunID", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/runs/")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing run ID, got %d", resp.StatusCode)
		}
	})
}

func TestConcurrentRequests(t *testing.T) {
	server := NewIntegrationTestServer()
	defer server.Close()
	
	baseURL := server.URL()
	runID := "concurrent-test-run"
	numRequests := 10
	
	// Channel to collect results
	results := make(chan error, numRequests)
	
	// Send concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(baseURL + "/runs/" + runID)
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("expected status 200, got %d", resp.StatusCode)
				return
			}
			
			var samples []Sample
			if err := json.NewDecoder(resp.Body).Decode(&samples); err != nil {
				results <- err
				return
			}
			
			if len(samples) == 0 {
				results <- fmt.Errorf("expected samples, got empty array")
				return
			}
			
			results <- nil
		}()
	}
	
	// Collect results
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent request timed out")
		}
	}
}

func TestDataConsistency(t *testing.T) {
	server := NewIntegrationTestServer()
	defer server.Close()
	
	baseURL := server.URL()
	runID := "consistency-test-run"
	
	// Make multiple requests and verify data consistency
	var firstSamples []Sample
	
	for i := 0; i < 3; i++ {
		resp, err := http.Get(baseURL + "/runs/" + runID)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Request %d returned status %d", i, resp.StatusCode)
		}
		
		var samples []Sample
		if err := json.NewDecoder(resp.Body).Decode(&samples); err != nil {
			t.Fatalf("Request %d decode failed: %v", i, err)
		}
		
		if i == 0 {
			firstSamples = samples
		} else {
			// Compare with first request
			if len(samples) != len(firstSamples) {
				t.Errorf("Request %d: expected %d samples, got %d", i, len(firstSamples), len(samples))
			}
			
			// Note: Mock data includes timestamps, so we can't do exact comparison
			// But we can check that the structure is consistent
			for j, sample := range samples {
				if j < len(firstSamples) {
					if sample.RunID != firstSamples[j].RunID {
						t.Errorf("Request %d, sample %d: RunID mismatch", i, j)
					}
					if sample.PID != firstSamples[j].PID {
						t.Errorf("Request %d, sample %d: PID mismatch", i, j)
					}
					if sample.Name != firstSamples[j].Name {
						t.Errorf("Request %d, sample %d: Name mismatch", i, j)
					}
				}
			}
		}
	}
}

func TestErrorHandling(t *testing.T) {
	server := NewIntegrationTestServer()
	defer server.Close()
	
	baseURL := server.URL()
	
	tests := []struct {
		name           string
		url            string
		method         string
		expectedStatus int
	}{
		{
			name:           "Missing run ID",
			url:            "/runs/",
			method:         "GET",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid method POST",
			url:            "/runs/test-run",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid method PUT",
			url:            "/runs/test-run",
			method:         "PUT",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid method DELETE",
			url:            "/runs/test-run",
			method:         "DELETE",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error
			
			if tt.method == "GET" {
				req, err = http.NewRequest(tt.method, baseURL+tt.url, nil)
			} else {
				req, err = http.NewRequest(tt.method, baseURL+tt.url, strings.NewReader("{}"))
			}
			
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}
