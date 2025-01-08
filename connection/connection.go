package connection

import (
	"crypto/tls"
	"errors"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
)

// GrpcConnectionConfig contains configuration options for the gRPC connection.
// This is primarily used to manage TLS and OAuth settings.
type GrpcConnectionConfig struct {
	TlsEnabled       bool
	ApiEndpoint      string
	CertificatePath  string
	AllowInsecureTls bool
	AuthToken        string
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

	// Next, if an OAuth token was provided, append that to our list
	// of dial options.
	if len(connectionConfig.AuthToken) > 0 {
		perRPC := oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(fetchToken(connectionConfig.AuthToken))}
		opts = append(opts, grpc.WithPerRPCCredentials(perRPC))
	}

	newConnection.ApiConnection, err =
		grpc.NewClient(connectionConfig.ApiEndpoint, opts...)

	if err != nil {
		return nil, err
	}

	return newConnection, nil
}

// fetchToken simulates a token lookup and omits the details of proper token
// acquisition. For examples of how to acquire an OAuth2 token, see:
// https://godoc.org/golang.org/x/oauth2
func fetchToken(authToken string) *oauth2.Token {
	return &oauth2.Token{
		AccessToken: authToken,
	}
}
