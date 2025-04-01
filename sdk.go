package lumenvox_go_sdk

import (
	"github.com/lumenvox/go-sdk/auth"
	"github.com/lumenvox/go-sdk/client"
	"github.com/lumenvox/go-sdk/connection"
)

// CreateConnection returns a gRPC connection object which can be used to create
// a client. It expects the endpoint of the API and various TLS settings, etc.
func CreateConnection(apiEndpoint string, tlsEnabled bool, certificatePath string, allowInsecureTls bool) (
	conn *connection.GrpcConnection, err error) {

	return client.CreateConnection(apiEndpoint, tlsEnabled, certificatePath, allowInsecureTls)
}

// CreateClient creates and initializes an SDK client using the provided gRPC
// connection, deployment ID, and authentication settings.
func CreateClient(conn *connection.GrpcConnection, deploymentId string, authSettings *auth.AuthSettings) (sdkClient *client.SdkClient) {

	return client.CreateSdkClient(conn, deploymentId, authSettings)
}
