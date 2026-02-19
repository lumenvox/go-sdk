// Package logging provides logger initialization and injection for the
// LumenVox SDK.
//
// The SDK uses Go's standard [log/slog] package for structured JSON logging.
// Call [CreateLogger] early in your application to set the log level and
// service name. If you need to integrate with an existing logging framework,
// use [SetLogger] to inject a custom [*slog.Logger].
//
// If neither function is called, the SDK falls back to a default logger
// that writes INFO-level JSON to stdout.
//
// [CreateLogger] or [SetLogger] should be called before any other SDK
// operations, as internal packages cache the logger on first use.
package logging
