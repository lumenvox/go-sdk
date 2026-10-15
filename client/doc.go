// Package client provides the SDK client type and settings builders for the
// LumenVox API.
//
// The primary type is [SdkClient], which is obtained by calling
// [github.com/lumenvox/go-sdk/v2.CreateClient]. It provides methods to create
// sessions and to build settings objects for various interaction types.
//
// # Sessions
//
// Call [SdkClient.NewSession] to open a bidirectional gRPC stream with the
// API. The returned [session.SessionObject] is used to create and manage
// interactions.
//
// # Settings Builders
//
// The client provides builder methods that construct protobuf settings
// objects with sensible defaults:
//
//   - [SdkClient.GetAudioConsumeSettings] — audio consumption mode and channel.
//   - [SdkClient.GetVadSettings] — voice activity detection parameters.
//   - [SdkClient.GetRecognitionSettings] — ASR recognition parameters.
//   - [SdkClient.GetNluSettings] — natural language understanding parameters.
//   - [SdkClient.GetNormalizationSettings] — text normalization and redaction.
//   - [SdkClient.GetAmdSettings] — answering machine detection parameters.
//   - [SdkClient.GetCpaSettings] — call progress analysis parameters.
//   - [SdkClient.GetTtsInlineSynthesisSettings] — text-to-speech voice configuration.
package client
