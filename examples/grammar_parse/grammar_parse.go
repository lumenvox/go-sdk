package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
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

	// Create the client
	client := lumenvoxSdk.CreateClient(conn, cfg.DeploymentId, nil)

	logger.Info("successfully created connection to LumenVox API!")

	///////////////////////
	// Session creation
	///////////////////////

	// Set audio configuration for session.
	audioConfig := session.AudioConfig{}

	// Create a new session.
	streamTimeout := 5 * time.Minute
	sessionObject, err := client.NewSession(streamTimeout, audioConfig)
	if err != nil {
		logger.Error("failed to create session",
			"error", err.Error())
		os.Exit(1)
	}

	///////////////////////
	// Create grammar parse interaction
	///////////////////////

	language := "en-US"
	inputText := "one two three four"

	// Configure grammar(s).
	grammars := []*api.Grammar{
		{ // Digits grammar
			GrammarLoadMethod: &api.Grammar_GrammarUrl{
				GrammarUrl: "http://assets.lumenvox.com/grammar/en/en_digits.grxml",
			},
			Label: &api.OptionalString{
				Value: "digits-grammar",
			},
		},
	}

	// Configure settings
	var grammarSettings *api.GrammarSettings = nil
	var generalInteractionSettings *api.GeneralInteractionSettings = nil
	parseTimeoutMs := &api.OptionalInt32{Value: 10000}

	// Create interaction.
	grammarParseInteraction, err := sessionObject.NewGrammarParse(language, inputText, grammars, grammarSettings,
		parseTimeoutMs, generalInteractionSettings)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := grammarParseInteraction.InteractionId
	logger.Info("received interactionId",
		"interactionId", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to become available.
	finalResults, err := grammarParseInteraction.GetFinalResults(10 * time.Second)
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
