package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/data"
)

func TestNew(t *testing.T) {
	agents := &data.AllAgents

	handler := New(agents)
	if handler == nil {
		t.Fatal("New() returned nil")
	}
}

func TestHandleHealth(t *testing.T) {
	agents := &data.AllAgents

	handler := New(agents)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleRoot(t *testing.T) {
	agents := &data.AllAgents

	handler := New(agents)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.HandleRoot(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestSetupRoutes(t *testing.T) {
	agents := &data.AllAgents

	handler := New(agents)
	router := handler.SetupRoutes()

	if router == nil {
		t.Fatal("SetupRoutes() returned nil")
	}
}
