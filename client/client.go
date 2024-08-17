package client

import (
    "github.com/lumenvox/go-sdk/connection"
    "github.com/lumenvox/go-sdk/session"
    "github.com/google/uuid"
    "time"
)

// SdkClient represents a client object with knowledge of API connectivity. It
// is primarily used to create session objects and settings objects.
type SdkClient struct {
    Connection   *connection.GrpcConnection
    DeploymentId string
}

// CreateSdkClient attempts to create a client object with the provided
// connection settings. apiEndpoint should contain the address of the
// lumenvox-api, and deploymentId should contain your deployment ID.
//
// Most users will want to enable TLS. To do this, tlsEnabled must be true.
// Depending on your environment, you may need to provide a root certificate
// using certificatePath. allowInsecureTls may be used to avoid this
// requirement, but this setting should not be used in production.
func CreateSdkClient(apiEndpoint string, tlsEnabled bool, certificatePath string, allowInsecureTls bool,
    deploymentId string) (client *SdkClient, err error) {

    client = &SdkClient{
        Connection:   nil,
        DeploymentId: deploymentId,
    }

    connectionConfig := connection.GrpcConnectionConfig{
        TlsEnabled:       tlsEnabled,
        ApiEndpoint:      apiEndpoint,
        CertificatePath:  certificatePath,
        AllowInsecureTls: allowInsecureTls,
    }

    client.Connection, err = connection.CreateNewConnection(connectionConfig)

    return client, err
}

// NewSession attempts to create a new session for the specified sdkClient. On
// success, it will return the new session object.
//
// streamTimeout controls the maximum duration of the stream, and audioConfig
// controls the audio configuration: streaming/batch, audio format, sample rate, etc.
func (client *SdkClient) NewSession(streamTimeout time.Duration, audioConfig session.AudioConfig) (
    newSession *session.SessionObject, err error) {

    operatorId := uuid.NewString()

    newSession, err = session.CreateNewSession(client.Connection.ApiConnection, streamTimeout, client.DeploymentId,
        audioConfig, operatorId)

    return newSession, err
}
