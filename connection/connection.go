package connection

import (
	"context"
	"crypto/tls"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"time"
)

// GrpcConnectionConfig contains configuration options for the gRPC connection.
// This is primarily used to manage TLS and related settings.
type GrpcConnectionConfig struct {
	TlsEnabled       bool
	ApiEndpoint      string
	CertificatePath  string
	AllowInsecureTls bool
	MaxMessageMb     int
}

// GrpcConnection contains the actual client connection object along with its
// configuration.
type GrpcConnection struct {
	ApiConnection    *grpc.ClientConn
	ConnectionConfig GrpcConnectionConfig
}

// CreateNewConnection accepts a GrpcConnectionConfig and uses it to create a new connection to the LumenVox API.
func CreateNewConnection(connectionConfig GrpcConnectionConfig) (newConnection *GrpcConnection, err error) {

	newConnection = &GrpcConnection{
		ApiConnection:    nil,
		ConnectionConfig: connectionConfig,
	}

	if connectionConfig.ApiEndpoint == "" {
		return nil, errors.New("empty endpoint")
	}

	// Handle the various dial options before initializing the client.
	var opts []grpc.DialOption = nil

	// First, handle TLS options, including SSL_VERIFY_PEER (insecure).
	// Append the result to our list of dial options.
	var creds credentials.TransportCredentials
	if connectionConfig.TlsEnabled {
		creds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: connectionConfig.AllowInsecureTls,
		})
	} else {
		creds = insecure.NewCredentials()
	}
	opts = append(opts, grpc.WithTransportCredentials(creds))

	// send a keepalive ping every 5 minutes
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time: 5 * time.Minute,
	}))

	if connectionConfig.MaxMessageMb < 1 {
		// Use 4MB if not specified.
		connectionConfig.MaxMessageMb = 4
	}

	// Add message size limits (default 4MB for both sending and receiving)
	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(connectionConfig.MaxMessageMb*1024*1024), // i.e. 4MB for receiving
		grpc.MaxCallSendMsgSize(connectionConfig.MaxMessageMb*1024*1024), // i.e. 4MB for sending
	))

	// Create the client
	newConnection.ApiConnection, err =
		grpc.NewClient(connectionConfig.ApiEndpoint, opts...)
	if err != nil {
		return nil, err
	}

	// Check the initial client state
	clientState := newConnection.ApiConnection.GetState()
	switch clientState {
	case connectivity.Idle:
		// The typical initial state. Call .Connect() to start a connection.
		newConnection.ApiConnection.Connect()
	case connectivity.Connecting:
		// The client is already establishing a connection. This is unexpected
		// here, but it may work.
	case connectivity.Ready:
		// The client is already fully connected. Return the new connection.
		return newConnection, nil
	case connectivity.TransientFailure:
		// The connection experienced a failure. In future releases, we may
		// attempt to recover from this. For now, this will be treated as a failure.
		return nil, errors.New("failed to connect: transient failure")
	case connectivity.Shutdown:
		// The connection is starting to shut down. Treat this as a failure.
		return nil, errors.New("failed to connect: shutdown")
	}

	// If we reach this point, either 1) the client was in the Idle state and we
	// called .Connect() or 2) the client was already in the Connecting state.
	// Wait through state changes until we error out or reach the Ready state.
	connectContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		if newConnection.ApiConnection.WaitForStateChange(connectContext, clientState) {
			// The state changed. Check the new state.
			clientState = newConnection.ApiConnection.GetState()
			switch clientState {
			case connectivity.Idle:
				// Somehow we got back to Idle. Call .Connect() again and continue.
				newConnection.ApiConnection.Connect()
			case connectivity.Connecting:
				// Connection establishing... keep waiting.
			case connectivity.Ready:
				// Connection established. Return the new connection.
				return newConnection, nil
			case connectivity.TransientFailure:
				// The connection experienced a failure. In future releases, we may
				// attempt to recover from this. For now, this will be treated as a failure.
				return nil, errors.New("failed to connect: transient failure")
			case connectivity.Shutdown:
				// The connection is starting to shut down. Treat this as a failure.
				return nil, errors.New("failed to connect: shutdown")
			}
		} else {
			// The context expired while waiting to connect. Return an error.
			return nil, errors.New("failed to connect: timeout")
		}
	}
}
