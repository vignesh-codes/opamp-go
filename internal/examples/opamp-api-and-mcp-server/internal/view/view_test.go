package view

import "testing"

func TestGetTopologyViewHTML(t *testing.T) {
	html := GetTopologyViewHTML()

	if html == "" {
		t.Fatal("GetTopologyViewHTML() returned empty string")
	}

	// Check for key HTML elements
	if len(html) < 100 {
		t.Error("HTML content seems too short")
	}

	// Check for essential elements
	checks := []string{
		"<!DOCTYPE html>",
		"<html>",
		"<head>",
		"<body>",
		"OpAMP Topology",
		"loadTopology",
	}

	for _, check := range checks {
		if !contains(html, check) {
			t.Errorf("HTML should contain '%s'", check)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
