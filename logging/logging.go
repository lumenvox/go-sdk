package logging

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	defaultLogger     = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger            *slog.Logger
	loggerInitialized = false
	loggerMu          sync.RWMutex
)

// init initializes the default logger.
func init() {
	// This is done, so that anyone requesting a logger before
	// it's initialized will get the default one, and knows to
	// request later, assuming someone calls CreateLogger
	logger = defaultLogger
}

// CreateLogger initializes and returns a new logger configured with the
// desired log level and service name.
func CreateLogger(levelStr string, serviceName string) *slog.Logger {

	logLevel, logLevelErr := GetLogLevel(levelStr)
	// Note: handling error after logging has been initialized below

	// Create a JSON logger
	newLogger := slog.New(slog.NewJSONHandler(os.Stdout,
		&slog.HandlerOptions{
			Level: logLevel, // log level from configuration
		})).With("service", serviceName)

	SetLogger(newLogger)

	returnedLogger, _ := GetLogger()
	if logLevelErr != nil {
		returnedLogger.Error(fmt.Sprintf("unable to get log level: %v", logLevelErr))
	}

	return returnedLogger
}

// GetLogLevel converts a string to slog.Level
func GetLogLevel(levelStr string) (slog.Level, error) {

	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, errors.New("invalid log level: use debug, info, warn, or error")
	}
}

// SetLogger allows customers to inject their own logger.
func SetLogger(customLogger *slog.Logger) {

	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = customLogger
	loggerInitialized = true
}

// GetLogger safely retrieves the current logger, and whether it was
// initialized, or is using the default.
func GetLogger() (*slog.Logger, bool) {

	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return logger, loggerInitialized
}
