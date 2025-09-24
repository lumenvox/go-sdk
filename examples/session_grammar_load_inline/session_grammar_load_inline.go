package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/auth"
	"github.com/lumenvox/go-sdk/config"
	"github.com/lumenvox/go-sdk/logging"
	"github.com/lumenvox/go-sdk/session"

	"github.com/lumenvox/protos-go/lumenvox/api"

	"os"
	"time"
)

func main() {

	///////////////////////
	// Client creation
	///////////////////////

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

	authSettings := &auth.AuthSettings{
		ClientId:    cfg.ClientId,
		SecretHash:  cfg.SecretHash,
		AuthHeaders: cfg.GetAuthHeaders(),
		Username:    cfg.Username,
		Password:    cfg.Password,
		AuthUrl:     cfg.AuthUrl,
	}

	if cfg.AuthUrl == "" || cfg.Username == "" || cfg.Password == "" || cfg.ClientId == "" || cfg.SecretHash == "" {
		authSettings = nil
	}

	// Create the client
	client := lumenvoxSdk.CreateClient(conn, cfg.DeploymentId, authSettings)

	logger.Info("successfully created connection to LumenVox API!")

	///////////////////////
	// Session creation
	///////////////////////

	// Set the audio configuration for the session. It's just text processing, so
	// the format should be set to NO_AUDIO_RESOURCE.
	audioConfig := session.AudioConfig{
		Format: api.AudioFormat_STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE,
	}

	// Create a new session.
	streamTimeout := 5 * time.Minute
	sessionObject, err := client.NewSession(streamTimeout, audioConfig)
	if err != nil {
		logger.Error("failed to create session",
			"error", err.Error())
		os.Exit(1)
	}

	///////////////////////
	// Create session grammar load request
	///////////////////////

	language := "en-US"
	grammarLabel := "digits-grammar"

	grammarBytes, err := os.ReadFile("examples/test_data/en_digits.grxml")
	if err != nil {
		logger.Error("reading grammar file",
			"error", err.Error())
		sessionObject.CloseSession()
		os.Exit(1)
	}
	grammarText := string(grammarBytes)

	// Configure settings
	var grammarSettings *api.GrammarSettings = nil

	// Create request.
	sessionGrammarLoadRequest, err := sessionObject.NewInlineLoadSessionGrammar(language, grammarLabel, grammarText,
		grammarSettings)
	if err != nil {
		logger.Error("failed to create request",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to become available.
	finalResults, err := sessionGrammarLoadRequest.GetFinalResults(10 * time.Second)
	if err != nil {
		logger.Error("waiting for final results",
			"error", err)
	} else {
		logger.Info("got final results",
			"finalResults", finalResults)
	}

	///////////////////////
	// Session close
	///////////////////////

	sessionObject.CloseSession()
}
