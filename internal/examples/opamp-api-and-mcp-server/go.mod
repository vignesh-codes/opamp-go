module github.com/open-telemetry/opamp-go/internal/examples/mcp-server

go 1.23.0

replace github.com/open-telemetry/opamp-go => ../../..

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/oklog/ulid/v2 v2.1.0
	github.com/open-telemetry/opamp-go v0.1.0
	github.com/open-telemetry/opamp-go/internal/examples v0.0.0-20251029170859-ad4ec0141946
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	google.golang.org/protobuf v1.36.7 // indirect
)
