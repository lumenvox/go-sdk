// Package auth provides authentication configuration for the LumenVox SDK.
//
// When connecting to an OAuth-protected LumenVox API endpoint, create an
// [AuthSettings] struct with your credentials and pass it to
// [github.com/lumenvox/go-sdk.CreateClient]. The SDK handles token retrieval
// and caching automatically; no manual token management is needed.
//
// If your LumenVox deployment does not use OAuth, pass nil for the auth
// settings parameter when creating a client.
package auth
