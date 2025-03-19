package session

import (
	"github.com/lumenvox/go-sdk/logging"

	"log/slog"
	"sync"
)

var (
	packageLogger     *slog.Logger // logger instance used by this package
	loggerInitialized bool         // whether the main logger was initialized
	loggerMu          sync.RWMutex // mutex to protect concurrent access to logger... vars
)

// getLogger returns a singleton instance of the logger. It ensures thread-safe
// initialization and reuse of the logger within this package.
func getLogger() *slog.Logger {

	loggerMu.Lock()
	defer loggerMu.Unlock()

	if loggerInitialized {
		return packageLogger
	}

	var init bool
	packageLogger, init = logging.GetLogger()
	if init {
		loggerInitialized = true
	}

	return packageLogger
}
