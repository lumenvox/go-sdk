package connection

import (
	"crypto/tls"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcConnectionConfig struct {
	TlsEnabled       bool
	ApiEndpoint      string
	CertificatePath  string
	AllowInsecureTls bool
}

type GrpcConnection struct {
	ApiConnection    *grpc.ClientConn
	ConnectionConfig GrpcConnectionConfig
}

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
