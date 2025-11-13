package main

import (
	"log"
	"os"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/mcp"
)

func main() {
	server := mcp.NewServer()

	if err := server.Run(); err != nil {
		log.Printf("Error running MCP server: %v", err)
		os.Exit(1)
	}
}
