package connection

import (
	"crypto/tls"
	"errors"
	"google.golang.org/grpc"
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

	newConnection.ApiConnection, err =
		grpc.NewClient(connectionConfig.ApiEndpoint, opts...)

	if err != nil {
		return nil, err
	}

	return newConnection, nil
}
