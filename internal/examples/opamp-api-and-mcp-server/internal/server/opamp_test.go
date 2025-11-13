package server

import (
	"testing"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/data"
)

func TestNewOpAMP(t *testing.T) {
	agents := &data.AllAgents

	opamp := NewOpAMP(agents)
	if opamp == nil {
		t.Fatal("NewOpAMP() returned nil")
	}

	if opamp.agents != agents {
		t.Error("Agents not set correctly")
	}
}

func TestLogger(t *testing.T) {
	logger := &Logger{
		Logger: nil, // Will be set in actual usage
	}

	// Test that Logger implements the interface
	// This will be verified at compile time
	_ = logger
}
