package lumenvox_go_sdk

import (
	"github.com/lumenvox/go-sdk/client"
	"github.com/lumenvox/go-sdk/connection"
)

// CreateConnection returns a gRPC connection object which can be used to create
// a client. It expects the endpoint of the API, various TLS settings, and an
// OAuth token.
func CreateConnection(apiEndpoint string, tlsEnabled bool, certificatePath string, allowInsecureTls bool,
	authToken string) (conn *connection.GrpcConnection, err error) {

	return client.CreateConnection(apiEndpoint, tlsEnabled, certificatePath, allowInsecureTls, authToken)
}

// CreateClient returns an SDK client object which can be used to access other
// LumenVox SDK functionality. It expects a GrpcConnection and the deployment
// ID.
func CreateClient(conn *connection.GrpcConnection, deploymentId string) (sdkClient *client.SdkClient) {

	return client.CreateSdkClient(conn, deploymentId)
}
