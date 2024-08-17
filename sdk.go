package lumenvox_go_sdk

import (
    "github.com/lumenvox/go-sdk/client"
)

// CreateClient returns an SDK client object which can be used to access other
// LumenVox SDK functionality.
func CreateClient(apiEndpoint string, tlsEnabled bool, certificatePath string, allowInsecureTls bool,

    deploymentId string) (sdkClient *client.SdkClient, err error) {

    sdkClient, err = client.CreateSdkClient(apiEndpoint, tlsEnabled, certificatePath, allowInsecureTls, deploymentId)

    return sdkClient, err
}

