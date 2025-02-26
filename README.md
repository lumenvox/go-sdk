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

### Client Creation

The first step in using the SDK is to create a connection to the API. This is
handled with the `CreateClient` function.
```go
apiEndpoint := "localhost:8280"
tlsEnabled := false
certificatePath := ""
allowInsecureTls := false
deploymentId := "00000000-0000-0000-0000-000000000000"
client, err := lumenvoxSdk.CreateClient(apiEndpoint, tlsEnabled, certificatePath,
    allowInsecureTls, deploymentId)
```

This is where connection details can be configured, including TLS options.

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
