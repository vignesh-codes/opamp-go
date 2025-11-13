package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/internal/examples/certs"
	"github.com/open-telemetry/opamp-go/internal/examples/mcp-server/internal/data"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server"
	servertypes "github.com/open-telemetry/opamp-go/server/types"
)

// OpAMP wraps the OpAMP server
type OpAMP struct {
	opampSrv  server.OpAMPServer
	agents    *data.Agents
	logger    types.Logger
	callbacks servertypes.Callbacks
}

// Logger implements types.Logger
type Logger struct {
	*log.Logger
}

func (l *Logger) Debugf(ctx context.Context, format string, v ...interface{}) {
	l.Printf("[DEBUG] "+format, v...)
}

func (l *Logger) Errorf(ctx context.Context, format string, v ...interface{}) {
	l.Printf("[ERROR] "+format, v...)
}

// NewOpAMP creates a new OpAMP server wrapper
func NewOpAMP(agents *data.Agents) *OpAMP {
	logger := &Logger{
		Logger: log.New(
			log.Default().Writer(),
			"[OPAMP] ",
			log.Default().Flags()|log.Lmsgprefix|log.Lmicroseconds,
		),
	}

	wrapper := &OpAMP{
		agents: agents,
		logger: logger,
	}

	// Set up callbacks using the local API structure
	wrapper.callbacks = servertypes.Callbacks{
		OnConnecting: func(request *http.Request) servertypes.ConnectionResponse {
			logger.Debugf(context.Background(), "OnConnecting called from %s", request.RemoteAddr)
			return servertypes.ConnectionResponse{
				Accept: true,
				ConnectionCallbacks: servertypes.ConnectionCallbacks{
					OnMessage:         wrapper.onMessage,
					OnConnectionClose: wrapper.onDisconnect,
				},
			}
		},
	}

	wrapper.opampSrv = server.New(logger)

	return wrapper
}

// Start starts the OpAMP server
func (srv *OpAMP) Start() {
	settings := server.StartSettings{
		Settings: server.Settings{
			Callbacks: srv.callbacks,
		},
		ListenEndpoint: getListenEndpoint(),
		HTTPMiddleware: otelhttp.NewMiddleware("/v1/opamp"),
		// TLS is optional - can be configured via environment variables if needed
		// For now, we run without TLS by default
	}

	tlsConfig, err := certs.CreateServerTLSConfig(
		certs.CaCert,
		certs.ServerCert,
		certs.ServerKey,
	)
	if err != nil {
		srv.logger.Debugf(context.Background(), "Could not load TLS config, working without TLS: %v", err.Error())
	} else {
		settings.TLSConfig = tlsConfig
	}

	if err := srv.opampSrv.Start(settings); err != nil {
		srv.logger.Errorf(context.Background(), "OpAMP server start fail: %v", err.Error())
		os.Exit(1)
	}
}

func getListenEndpoint() string {
	if endpoint := os.Getenv("OPAMP_LISTEN_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "0.0.0.0:4320"
}

// Stop stops the OpAMP server
func (srv *OpAMP) Stop() {
	srv.opampSrv.Stop(context.Background())
}

func (srv *OpAMP) onDisconnect(conn servertypes.Connection) {
	srv.agents.RemoveConnection(conn)
}

func (srv *OpAMP) onMessage(ctx context.Context, conn servertypes.Connection, msg *protobufs.AgentToServer) *protobufs.ServerToAgent {
	// Start building the response.
	response := &protobufs.ServerToAgent{}

	var instanceId data.InstanceId
	// In local code, InstanceUid is []byte
	instanceUid := msg.InstanceUid

	if len(instanceUid) == 26 {
		// This is an old-style ULID (26 bytes as string).
		u, err := ulid.Parse(string(instanceUid))
		if err != nil {
			srv.logger.Errorf(ctx, "Cannot parse ULID %s: %v", string(instanceUid), err)
			return response
		}
		instanceId = data.InstanceId(u.Bytes())
	} else if len(instanceUid) == 16 {
		// This is a 16 byte, new style UID.
		instanceId = data.InstanceId(instanceUid)
	} else {
		srv.logger.Errorf(ctx, "Invalid length of msg.InstanceUid: %d", len(instanceUid))
		return response
	}

	agent := srv.agents.FindOrCreateAgent(instanceId, conn)
	srv.logger.Debugf(ctx, "Agent found/created: instanceId=%x, total agents: %d", instanceId, len(srv.agents.GetAllAgentsReadonlyClone()))

	// Process the status report and continue building the response.
	agent.UpdateStatus(msg, response)

	if msg.ConnectionSettingsStatus != nil {
		srv.logger.Debugf(ctx, "Connection settings for instance %x %s (err=%s) hash=%x", instanceId, msg.ConnectionSettingsStatus.Status.String(), msg.ConnectionSettingsStatus.ErrorMessage, msg.ConnectionSettingsStatus.LastConnectionSettingsHash)
	}

	// Send the response back to the Agent.
	return response
}
