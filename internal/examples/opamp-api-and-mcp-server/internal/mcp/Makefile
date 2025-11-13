.PHONY: build clean

build:
	@echo "Building MCP server..."
	@cd cmd/mcp-server && go build -o ../../mcp-server .
	@echo "Binary created at: mcp-server"

clean:
	@echo "Cleaning..."
	@rm -f mcp-server
	@echo "Done"

test:
	@echo "Running tests..."
	@go test ./...


# "opamp-fleet": {
#       "command": "/home/vws/Desktop/vws-codes/projects/PaaS-On-Kubernetes/opamp-go/mcp-server/mcp-server",
#       "args": [],
#       "env": {
#         "OPAMP_API_URL": "http://localhost:9081"
#       }
#     },