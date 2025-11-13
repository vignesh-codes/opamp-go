package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/api"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/data"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/server"
)

func main() {
	// Get port from environment, default to 9081
	port := os.Getenv("PORT")
	if port == "" {
		port = "9081"
	}

	log.Printf("Starting OpAMP Server with Topology API...")
	log.Printf("API Port: %s", port)

	// Initialize agents
	agents := data.AllAgents

	// Initialize OpAMP server
	opampServer := server.NewOpAMP(agents)

	// Start OpAMP server (for agent connections)
	opampServer.Start()
	opampEndpoint := os.Getenv("OPAMP_LISTEN_ENDPOINT")
	if opampEndpoint == "" {
		opampEndpoint = "0.0.0.0:4320"
	}
	log.Printf("OpAMP server started on %s", opampEndpoint)

	// Initialize API handler
	apiHandler := api.New(agents)
	router := apiHandler.SetupRoutes()

	// Add CORS headers for development
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	log.Printf("API Server running on http://localhost:%s", port)
	log.Printf("API Documentation: http://localhost:%s/", port)
	log.Printf("Topology View: http://localhost:%s/api/topology/view", port)
	log.Printf("Topology JSON: http://localhost:%s/api/topology", port)
	log.Printf("Agents endpoint: http://localhost:%s/api/agents", port)

	// Start API server in a goroutine
	go func() {
		if err := http.ListenAndServe(":"+port, corsHandler(router)); err != nil {
			log.Fatalf("API server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	log.Println("Shutting down...")
	opampServer.Stop()
	log.Println("Shutdown complete")
}
