// Package lumenvox_go_sdk provides a Go client for the LumenVox AI API.
//
// This is the main entry point for the SDK. It exposes factory functions for
// creating connections and clients that interact with the LumenVox API over
// gRPC.
//
// # Getting Started
//
// A typical usage flow looks like this:
//
//  1. Create a gRPC connection with [CreateConnection].
//  2. Create a client with [CreateClient], optionally providing [auth.AuthSettings] for OAuth.
//  3. Open a session with client.NewSession.
//  4. Create interactions (ASR, transcription, TTS, NLU, etc.) on the session.
//  5. Stream audio, consume results, and close the session when finished.
//
// # Sub-packages
//
//   - [github.com/lumenvox/go-sdk/v2/auth] — authentication settings for OAuth-protected endpoints.
//   - [github.com/lumenvox/go-sdk/v2/client] — the SDK client type and settings builders.
//   - [github.com/lumenvox/go-sdk/v2/config] — optional convenience utilities for loading configuration from INI files.
//   - [github.com/lumenvox/go-sdk/v2/logging] — logger initialization and injection.
//   - [github.com/lumenvox/go-sdk/v2/session] — session and interaction lifecycle management.
package lumenvox_go_sdk
