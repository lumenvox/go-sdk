# LumenVox Go SDK

The LumenVox Go SDK provides a Go client for the [LumenVox AI API](https://developer.lumenvox.com). It handles gRPC connectivity, session management, audio streaming, and result consumption so you can focus on building voice-enabled applications.

## Supported Interactions

| Category | Interaction Types |
|---|---|
| Speech Recognition | ASR, Transcription (streaming, batch, continuous) |
| Speech Synthesis | TTS (inline text, SSML URL) |
| Language Understanding | NLU (summarization, translation), Sentiment Analysis |
| Audio Analysis | Answering Machine Detection (AMD), Call Progress Analysis (CPA), Call Screening, Speaker Diarization, Language Identification |
| Text Processing | Normalization, Redaction, Grammar Parsing, Intent Detection (Neuron) |

## Requirements

- Go 1.26 or later
- Access to a LumenVox API endpoint (SaaS or on-prem)

## Installation

```bash
go get github.com/lumenvox/go-sdk
```

## Quick Start

Each example in the `examples/` directory is a self-contained Go module. To run one:

```bash
cd examples/transcription_streaming_vad
go run .
```

Before running examples, copy and edit the configuration file at the project root:

```bash
cp config_values.ini config_values.ini.bak
# Edit config_values.ini with your API endpoint, deployment ID, and credentials
```

## Usage

### 1. Initialize the Logger

```go
import "github.com/lumenvox/go-sdk/logging"

logger := logging.CreateLogger("info", "my-app")
```

Call this before any other SDK operations. The SDK uses Go's `log/slog` for structured JSON logging. To inject your own logger, use `logging.SetLogger()`.

### 2. Create a Connection and Client

```go
import (
    lumenvoxSdk "github.com/lumenvox/go-sdk"
    "github.com/lumenvox/go-sdk/auth"
)

// Create a gRPC connection
conn, err := lumenvoxSdk.CreateConnection(
    "api.example.com:443", // API endpoint
    true,                  // TLS enabled
    "",                    // certificate path (empty = system root CAs)
    false,                 // allow insecure TLS
)
if err != nil {
    logger.Error("connection failed", "error", err)
    os.Exit(1)
}

// Create a client (pass nil for authSettings if OAuth is not required)
client := lumenvoxSdk.CreateClient(conn, "your-deployment-id", nil)
```

A single connection can be shared across multiple clients.

### 3. Open a Session

```go
import (
    "github.com/lumenvox/go-sdk/session"
    "github.com/lumenvox/protos-go/lumenvox/api"
)

audioConfig := session.AudioConfig{
    Format:     api.AudioFormat_STANDARD_AUDIO_FORMAT_ULAW,
    SampleRate: 8000,
    IsBatch:    false,
}

sessionObject, err := client.NewSession(5*time.Minute, audioConfig)
if err != nil {
    logger.Error("session creation failed", "error", err)
    os.Exit(1)
}
defer sessionObject.CloseSession()
```

The `streamTimeout` parameter sets the maximum duration of the session. For sessions that do not require inbound audio (e.g. TTS, NLU, normalization), set the audio format to `NO_AUDIO_RESOURCE`:

```go
audioConfig := session.AudioConfig{
    Format: api.AudioFormat_STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE,
}
```

### 4. Run an Interaction

This example creates a streaming transcription with VAD (voice activity detection):

```go
// Build settings
audioConsumeSettings, _ := client.GetAudioConsumeSettings(
    0, // audio channel
    api.AudioConsumeSettings_AUDIO_CONSUME_MODE_STREAMING,
    api.AudioConsumeSettings_STREAM_START_LOCATION_STREAM_BEGIN,
    nil, nil,
)
vadSettings := client.GetVadSettings(true, 5000, 800, nil,
    api.VadSettings_NOISE_REDUCTION_MODE_ENABLED, nil, nil, nil, nil, nil)
recognitionSettings := client.GetRecognitionSettings(10000, false, nil, nil, nil)

// Create the interaction
transcription, err := sessionObject.NewTranscription(
    "en-us", nil, nil,
    audioConsumeSettings, nil, vadSettings, recognitionSettings,
    "", "", "", nil,
)
if err != nil {
    logger.Error("failed to create transcription", "error", err)
    os.Exit(1)
}

// Stream audio
audioData, _ := os.ReadFile("path/to/audio.ulaw")
sessionObject.AddAudio(audioData)

// Wait for results
err = transcription.WaitForFinalResults(30 * time.Second)
if err != nil {
    logger.Error("timed out waiting for results", "error", err)
    os.Exit(1)
}

results, err := transcription.GetFinalResults(5 * time.Second)
if err != nil {
    logger.Error("error getting results", "error", err)
} else {
    fmt.Println("Transcription:", results)
}
```

## Authentication

LumenVox SaaS endpoints are protected by OAuth. To authenticate, create an `AuthSettings` struct and pass it when creating the client:

```go
authSettings := &auth.AuthSettings{
    AuthUrl:     "https://cognito-idp.us-east-1.amazonaws.com/",
    ClientId:    "your-client-id",
    SecretHash:  "your-secret-hash",
    Username:    "your-username",
    Password:    "your-password",
    AuthHeaders: map[string]string{
        "Content-Type": "application/x-amz-json-1.1",
        "X-Amz-Target": "AWSCognitoIdentityProviderService.InitiateAuth",
    },
}

client := lumenvoxSdk.CreateClient(conn, deploymentId, authSettings)
```

The SDK handles token acquisition and caching automatically. Tokens are refreshed before expiry, so no manual token management is needed.

> **Important:** Keep your credentials secure. If you believe they have been compromised, contact LumenVox support to rotate them.

On-prem deployments that do not use OAuth should pass `nil` for the auth settings parameter.

## Configuration

The `config` package provides an optional helper for loading settings from INI files:

```go
import "github.com/lumenvox/go-sdk/config"

cfg, err := config.GetConfigValues("../../config_values.ini")
```

This is a convenience — you are free to configure the SDK using any method. All SDK functions accept plain Go types, not config objects.

Settings can also be overridden with environment variables using the `LUMENVOX_GO_SDK__` prefix:

```bash
export LUMENVOX_GO_SDK__API_ENDPOINT="api.example.com:443"
export LUMENVOX_GO_SDK__DEPLOYMENT_ID="your-deployment-id"
```

See `config_values.ini` for the full list of supported keys.

## Package Overview

| Package | Import Path | Description |
|---|---|---|
| Root | `github.com/lumenvox/go-sdk` | Entry point — `CreateConnection`, `CreateClient` |
| auth | `github.com/lumenvox/go-sdk/auth` | `AuthSettings` struct for OAuth configuration |
| client | `github.com/lumenvox/go-sdk/client` | `SdkClient` type, session creation, settings builders |
| config | `github.com/lumenvox/go-sdk/config` | Optional INI config loader |
| logging | `github.com/lumenvox/go-sdk/logging` | Logger initialization and injection |
| session | `github.com/lumenvox/go-sdk/session` | Session lifecycle, interaction objects, audio management |

## Examples

The `examples/` directory contains 32 runnable examples covering all supported interaction types:

| Example | Description |
|---|---|
| `open_and_close_session` | Minimal example — connect, open a session, close it |
| `simple_oauth` | OAuth authentication flow |
| `transcription_streaming_vad` | Streaming transcription with voice activity detection |
| `transcription_batch` | Batch transcription of a complete audio file |
| `transcription_batch_with_alias_lexicon` | Batch transcription with alias lexicon |
| `transcription_batch_with_phrase_list` | Batch transcription with phrase list |
| `transcription_streaming_vad_continuous` | Continuous streaming transcription |
| `transcription_streaming_vad_early_finalize` | Streaming transcription with early finalization |
| `transcription_streaming_vad_enhanced` | Streaming transcription with enhanced model |
| `transcription_streaming_vad_with_partials` | Streaming transcription with partial results |
| `transcription_streaming_vad_with_srt` | Transcription with SRT subtitle generation |
| `transcription_streaming_vad_with_vtt` | Transcription with VTT subtitle generation |
| `transcription_normalization_redaction` | Transcription with text normalization and redaction |
| `asr_streaming_vad` | ASR with grammar-based recognition |
| `asr_streaming_vad_early_finalize` | ASR with early interaction finalization |
| `tts_inline` | Text-to-speech from inline text |
| `tts_inline_with_partial_streaming` | TTS with partial audio streaming |
| `tts_uri` | Text-to-speech from an SSML URL |
| `normalization` | Text normalization |
| `sentiment_analysis` | Sentiment analysis |
| `nlu_sentiment_analysis` | NLU-based sentiment analysis |
| `nlu_summarization` | Text summarization |
| `grammar_parse` | Grammar-based text parsing |
| `session_grammar_load_inline` | Load a grammar into a session (inline) |
| `session_grammar_load_uri` | Load a grammar into a session (URI) |
| `amd` | Answering machine detection |
| `call_screening` | Call screening |
| `call_screening_grammar` | Call screening with grammar |
| `cpa` | Call progress analysis |
| `diarization` | Speaker diarization |
| `language_id` | Language identification |
| `neuron` | Intent detection |

## Troubleshooting

**Connection failures:** Verify your API endpoint is reachable and TLS settings are correct. Use `allowInsecureTls: true` only for development.

**Authentication errors:** Ensure all six auth fields (`AuthUrl`, `ClientId`, `SecretHash`, `Username`, `Password`, `AuthHeaders`) are set. The SDK will return an error on session creation if token acquisition fails.

**Verbose logging:** Set the environment variable `LUMENVOX_GO_SDK__ENABLE_VERBOSE_LOGGING=true` or set `session.EnableVerboseLogging = true` in code to enable detailed SDK logging.

## License

See [LICENSE](LICENSE) for details.
