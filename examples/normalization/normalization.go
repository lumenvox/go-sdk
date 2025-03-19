package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/config"
	"github.com/lumenvox/go-sdk/logging"
	"github.com/lumenvox/go-sdk/session"

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

	var accessToken string // Not assigned for now

	// Create connection. The connection should generally be reused when
	// you are creating multiple clients.
	conn, err := lumenvoxSdk.CreateConnection(
		cfg.ApiEndpoint,
		cfg.EnableTls,
		cfg.CertificatePath,
		cfg.AllowInsecureTls,
		accessToken,
	)
	if err != nil {
		logger.Error("failed to create connection",
			"error", err)
		os.Exit(1)
	}

	// Create the client
	client := lumenvoxSdk.CreateClient(conn, cfg.DeploymentId)

	logger.Info("successfully created connection to LumenVox API!")

	///////////////////////
	// Session creation
	///////////////////////

	// Set audio configuration for session. It's just text processing, so leaving it empty is fine.
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
	// Create normalization interaction
	///////////////////////

	language := "en-US"

	// Configure normalization settings.
	enableCapitalization := true
	normalizationSettings := client.GetNormalizationSettings(false, enableCapitalization, false, false, false, nil)

	textToNormalize := "recorded books presents an unabridged recording of the great gatsby by f scott" +
		" fitzgerald narrated by frank muller chapter one in my younger and more vulnerable years my father" +
		" gave me some advice that i've been turning over in my mind ever since whenever you feel like criticizing" +
		" anyone he told me just remember that all the people in this world haven't had the advantages that you've" +
		" had he didn't say any more but we've always been unusually communicative and a reserved way and i" +
		" understood that he meant a great deal more than that in consequence i am inclined to reserve all judgments" +
		" the habit that has opened up many curious natures to me and also made me the victim of not a few veteran" +
		" wars the abnormal mind is quick to detect and detach itself to this quality when it appears in a normal person"

	// Create interaction.
	normalizationInteraction, err := sessionObject.NewNormalization(language, textToNormalize,
		normalizationSettings, nil)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := normalizationInteraction.InteractionId
	logger.Info("received interactionId",
		"interactionId", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := normalizationInteraction.GetFinalResults(10 * time.Second)
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
