package prometheus

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	s := New("")
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.baseURL != "http://localhost:9090" {
		t.Errorf("Expected default baseURL 'http://localhost:9090', got '%s'", s.baseURL)
	}

	customURL := "http://prometheus:9090"
	s2 := New(customURL)
	if s2.baseURL != customURL {
		t.Errorf("Expected baseURL '%s', got '%s'", customURL, s2.baseURL)
	}
}

func TestQuery(t *testing.T) {
	// Create a mock Prometheus server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			t.Errorf("Expected path '/api/v1/query', got '%s'", r.URL.Path)
		}

		response := `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"receiver": "otlp"},
						"value": [1234567890, "100"]
					}
				]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	s := New(server.URL)
	result, err := s.Query("test_query")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
	}

	if len(result.Data.Result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result.Data.Result))
	}

	metric := result.Data.Result[0].Metric["receiver"]
	if metric != "otlp" {
		t.Errorf("Expected receiver 'otlp', got '%s'", metric)
	}
}

func TestQueryError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	s := New(server.URL)
	_, err := s.Query("test_query")
	if err == nil {
		t.Fatal("Expected error for 500 status, got nil")
	}
}
