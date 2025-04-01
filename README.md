# LumenVox Go SDK

The LumenVox Go SDK is designed to provide a convenient interface for
interacting with our LumenVox API.

The SDK currently includes support for:
  - ASR interactions
  - Transcription interactions (including continuous transcriptions)
  - TTS interactions
  - Text normalization interactions

We plan to support grammar parse interactions in the near future.

## Quick Start

The `examples` folder contains multiple example scripts for various interaction
types. With the repo cloned to your local machine, you can run an example from
the root of this project:
```shell
go run examples/1234_streaming_vad_asr/1234_streaming_vad_asr.go
```

Browsing through the example code can be helpful for understanding the design of
the SDK.

## Design Overview

The majority of the functionality offered by the LumenVox API is wrapped up in
**interactions**. There are multiple types of interactions for various
operations, including but not limited to:
  - Automatic Speech Recognition (ASR)
  - Transcription
  - Text-to-speech (TTS)
  - Text Normalization

Every interaction is the child of a **session**. Conceptually, a session can be
likened to a phone call. Over the course of a session, multiple interactions can
be processed, including concurrent interactions. If the interactions for a
session will include speech recognition (ASR or Transcription), the session will
also include an audio stream.

Sessions are implemented as gRPC streams, so each session is the child of a
**client**. The client controls the gRPC connection to the LumenVox API. 

## Writing an Application

### Configuration Values

Helper functions are included to allow the use of a configuration file. An
example file named `config_values.ini` has been provided. In your own
environment, this may be moved or renamed as needed.

Any values in the `.ini` file will override the default values. Additionally,
the values from the `.ini` file can be overridden with the use of environment
variables.

The list of supported values is shown in the `config_values.ini` file. The
environment variables have the same name with the addition of a prefix, as
described below.

> Note: environment variable names should have a `LUMENVOX_GO_SDK__` prefix
> to avoid conflict with other environment variables. For example, to
> specify the API_ENDPOINT, use the `LUMENVOX_GO_SDK__API_ENDPOINT`
> environment variable.

Users may choose to use the .ini file, environment variables, defaults, or
simply use hard-coded values in their files. We recommend either file-based
or environment-variable-based approaches.

### Client Creation

The first step in using the SDK is to create a connection to the API. This is
handled with the `CreateClient` function, as shown below (using the config
helpers). The connection can then be used to create the client you will use
to manage sessions.
```go
// Get SDK configuration
cfg, err := config.GetConfigValues("./config_values.ini")
if err != nil {
    tmpLogger, _ := logging.GetLogger() // get default logger
    tmpLogger.Error("unable to get config",
        "error", err)
    os.Exit(1)
}

logger := logging.CreateLogger(cfg.LogLevel, "lumenvox-go-sdk")

// Create connection. The connection should generally be reused when
// you are creating multiple clients.
conn, err := lumenvoxSdk.CreateConnection(
    cfg.ApiEndpoint,
    cfg.EnableTls,
    cfg.CertificatePath,
    cfg.AllowInsecureTls,
)
if err != nil {
    logger.Error("failed to create connection",
        "error", err)
    os.Exit(1)
}

// Create the client
client := lumenvoxSdk.CreateClient(conn, cfg.DeploymentId, nil)
```

### Authentication Token Handling

If connecting to LumenVox' SaaS APIs, you will be provided credentials allowing
access to the services. These can be configured in the `config_values.ini` file
as described here (or in corresponding environment variables):

| Setting      | Value                                                                                                 |
|--------------|-------------------------------------------------------------------------------------------------------|
| USERNAME     | "<your-username>"                                                                                     |
| PASSWORD     | "<your-password>"                                                                                     |
| CLIENT_ID    | "<your-client-id>"                                                                                    |
| SECRET_HASH  | "<your-secret-hash>"                                                                                  |
| AUTH_HEADERS | "Content-Type=application/x-amz-json-1.1,X-Amz-Target=AWSCognitoIdentityProviderService.InitiateAuth" |
| AUTH_URL     | "https://cognito-idp.us-east-1.amazonaws.com/"                                                        |

These key/value pairs should be in the form of `<key> = <value>`, for example:

```ini
USERNAME = "testuser"
```

> NOTE: It is important to understand that these credentials are secret and
> you should protect them and not share them with anyone. If you believe
> these credentials may have become compromised at any time, please reach out
> to our support team, who can generate new ones and revoke the compromised
> one(s).

Once you have defined these values, your code can then use them to acquire
OAuth tokens automatically, when needed. There is a built-in caching mechanism
to avoid making excessive calls to the OAuth service, and we encourage users
to utilize this caching method (which is automatically enabled) 

The `simple_oauth` example shows how these can be utilized, when needed. Also
note that it is best practice to enable TLS when using OAuth tokens. This is
a hard requirement when working with the LumenVox SaaS systems, but is in
general good practice if you are implementing your own.

Finally, the OAuth implementation was designed to work with the LumenVox
SaaS systems, and make connectivity to those easy for users, however it was
also written in a way in which users who wish to implement their own token
authorization within their own managed systems, may be able to use much of
the same code, however implementation of those methods is outside the scope
of what is supported. There are many online references available that can
offer better advise in how to implement your own authorization if needed. 

In general, automated token acquisition and checking can easily be added
by simply adding the definition shown below (from the `simple_oauth`
example):

```go
authSettings := &auth.AuthSettings{
    ClientId:    cfg.ClientId,
    SecretHash:  cfg.SecretHash,
    AuthHeaders: cfg.GetAuthHeaders(),
    Username:    cfg.Username,
    Password:    cfg.Password,
    AuthUrl:     cfg.AuthUrl,
}

// Create the client
client := lumenvoxSdk.CreateClient(conn, cfg.DeploymentId, authSettings)
```

The SDK will attempt to obtain a valid token when connecting to the LumenVox
SaaS system and will report any errors encountered. If the token expires
after its expiration period (which may vary!) then a new token will be
automatically requested and cached, allowing multiple clients to be created
throughout the life of the application, if this is needed.

### Session Creation

Once you have a client from the previous step, you need to create a session. This
is handled with the `NewSession` function.
```go
streamTimeout := 5 * time.Minute
audioConfig := session.AudioConfig{
    Format:     api.AudioFormat_STANDARD_AUDIO_FORMAT_ULAW,
    SampleRate: 8000,
    IsBatch:    false,
}
sessionObject, err := client.NewSession(streamTimeout, audioConfig)
```

`streamTimeout` represents a cap on the total length of the session, and
`audioConfig` controls the configuration of the audio stream. In this example,
the stream has been configured for streaming ULAW audio at 8kHz. Multiple audio
formats are supported at multiple sample rates, and both batch mode and streaming
mode are supported.

If you do not expect to stream any audio (for example, if you're only using TTS
and/or text normalization), an empty `audioConfig` is fine:
```go
streamTimeout := 5 * time.Minute
audioConfig := session.AudioConfig{}
sessionObject, err := client.NewSession(streamTimeout, audioConfig)
```

### Audio Stream Management

To add audio to the session, you can use the `AddAudio` function. If the audio
stream has been configured for batch mode, this function will block until all
the audio has been sent to the API. Otherwise, it will add the audio to an
internal queue (to be streamed in the background) and return immediately.
```go
audioFilePath := "./examples/test_data/1234.ulaw"
audioData, err = os.ReadFile(audioFilePath)

sessionObject.AddAudio(audioData)
```

### Interaction Management

There are multiple functions to create the various types of interactions. These
include:
  - `NewAsr`
  - `NewTranscription`
  - `NewNormalization`
  - `NewInlineTts`
  - `NewUrlTts`

Each of these functions returns an object representing the new interaction:
  - `AsrInteractionObject`
  - `TranscriptionInteractionObject`
  - `NormalizationInteractionObject`
  - `TtsInteractionObject`

Each interaction type offers different functions to manage processing and
results.

Most interactions accept settings objects during creation. These can be obtained
from various functions offered by the client:
  - `GetAudioConsumeSettings`
  - `GetNormalizationSettings`
  - `GetVadSettings`
  - `GetRecognitionSettings`
  - `GetTtsInlineSynthesisSettings`

#### AsrInteractionObject

The `AsrInteractionObject` represents an ASR interaction. The following functions
are available:
  - `WaitForBeginProcessing`
  - `WaitForBargeIn`
  - `WaitForEndOfSpeech`
  - `WaifForBargeInTimeout`
  - `WaitForFinalResults`
  - `WaitForNextResult`
  - `GetPartialResult`
  - `GetFinalResults`

To manually end an ASR interaction, whether you need to perform an early exit or
you just don't have VAD enabled, the `FinalizeInteraction` function is available
as a member function of the session object.

#### TranscriptionInteractionObject

The `TranscriptionInteractionObject` represents a transcription interaction. The
following functions are available:
- `WaitForBeginProcessing`
- `WaitForBargeIn`
- `WaitForEndOfSpeech`
- `WaifForBargeInTimeout`
- `WaitForFinalResults`
- `WaitForNextResult`
- `GetPartialResult`
- `GetFinalResults`

To manually end a transcription interaction, whether you need to perform an early
exit or you just don't have VAD enabled, the `FinalizeInteraction` function is
available as a member function of the session object.

#### NormalizationInteractionObject

The `NormalizationInteractionObject` represents a normalization interaction. The
following functions are available:
- `WaitForFinalResults`
- `GetFinalResults`

#### TtsInteractionObject

The `TtsInteractionObject` represents an TTS interaction. The following functions
are available:
- `WaitForFinalResults`
- `GetFinalResults`

Once the synthesis is complete, the audio can be retrieved with `PullTtsAudio`,
a member function of the session object.

### Session Cleanup

To close a session, the `CloseSession` function is available:
```go
sessionObject.CloseSession()
```
