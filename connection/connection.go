package connection

import (
	"crypto/tls"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GrpcConnectionConfig contains configuration options for the gRPC connection.
// This is primarily used to manage TLS settings.
type GrpcConnectionConfig struct {
	TlsEnabled       bool
	ApiEndpoint      string
	CertificatePath  string
	AllowInsecureTls bool
}

// GrpcConnection contains the actual client connection object along with its
// configuration.
type GrpcConnection struct {
	ApiConnection    *grpc.ClientConn
	ConnectionConfig GrpcConnectionConfig
}

// CreateNewConnection accepts a GrpcConnectionConfig and uses it to create a
// new connection to the lumenvox API.
func CreateNewConnection(connectionConfig GrpcConnectionConfig) (newConnection *GrpcConnection, err error) {

	newConnection = &GrpcConnection{
		ApiConnection:    nil,
		ConnectionConfig: connectionConfig,
	}

	if connectionConfig.ApiEndpoint == "" {
		return nil, errors.New("empty endpoint")
	}

	var creds credentials.TransportCredentials
	if connectionConfig.TlsEnabled {
		creds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: connectionConfig.AllowInsecureTls,
		})
	} else {
		creds = insecure.NewCredentials()
	}

	newConnection.ApiConnection, err = grpc.NewClient(connectionConfig.ApiEndpoint, grpc.WithTransportCredentials(creds))

	if err != nil {
		return nil, err
	}

	return newConnection, nil
}
