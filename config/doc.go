// Package config provides optional convenience utilities for loading SDK
// configuration from INI files.
//
// This package is not required to use the SDK. All core SDK functions accept
// plain Go types (strings, booleans, etc.), so you are free to source your
// configuration from any mechanism — environment variables, secret managers,
// or your own configuration library.
//
// # INI File Format
//
// Call [GetConfigValues] with the path to an INI file to load settings. The
// returned [ConfigValues] struct contains fields for all connection, audio,
// and authentication parameters used by the SDK examples.
//
// # Environment Variable Overrides
//
// Any field in [ConfigValues] can be overridden by setting an environment
// variable with the prefix LUMENVOX_GO_SDK__ followed by the field name in
// SCREAMING_SNAKE_CASE. For example, LUMENVOX_GO_SDK__API_ENDPOINT overrides
// the api_endpoint INI key.
package config
