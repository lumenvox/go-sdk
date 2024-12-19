package lumenvox_go_sdk

import (
    "github.com/lumenvox/go-sdk/client"
)

// CreateClient returns an SDK client object which can be used to access other
// LumenVox SDK functionality. It expects the endpoint of the API, various TLS
// settings, the deployment ID, and an OAuth token.
func CreateClient(apiEndpoint string, tlsEnabled bool, certificatePath string, allowInsecureTls bool,
	deploymentId string, authToken string) (sdkClient *client.SdkClient, err error) {

	sdkClient, err =
		client.CreateSdkClient(apiEndpoint, tlsEnabled, certificatePath, allowInsecureTls, deploymentId, authToken)

	return sdkClient, err
}
