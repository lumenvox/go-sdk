package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/config"
	"github.com/lumenvox/go-sdk/logging"
	"github.com/lumenvox/go-sdk/lumenvox/api"
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
	// Create NLU interaction
	///////////////////////

	language := "en-US"

	// Configure NLU settings.
	var summarizationBulletPoints int32 = 3
	var summarizationWords int32 = 0
	translateFromLanguage := ""
	translateToLanguage := ""
	enableLanguageDetect := false
	enableTopicDetect := false
	detectOutcomeType := api.NluSettings_UNDEFINED
	enableSentimentAnalysis := false
	var requestTimeoutMs *api.OptionalInt32 = nil
	nluSettings := client.GetNluSettings(summarizationBulletPoints, summarizationWords,
		translateFromLanguage, translateToLanguage, enableLanguageDetect, enableTopicDetect,
		detectOutcomeType, enableSentimentAnalysis, requestTimeoutMs)

	inputText := "LumenVox is a company specializing in speech recognition and voice biometrics technology. Their" +
		" solutions are designed to enhance customer interactions by enabling automated speech recognition (ASR) and" +
		" voice-based authentication in various applications, such as call centers and interactive voice response" +
		" (IVR) systems. LumenVox focuses on providing secure, accurate, and scalable voice solutions to streamline" +
		" processes and improve user experiences. Their technology is often used in industries like finance," +
		" healthcare, and telecommunications to provide faster and more secure services by leveraging natural voice" +
		" interactions."

	// Create interaction.
	var generalInteractionSettings *api.GeneralInteractionSettings = nil
	nluInteraction, err := sessionObject.NewNlu(language, inputText, nluSettings, generalInteractionSettings)
	if err != nil {
		logger.Error("failed to create NLU interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := nluInteraction.InteractionId
	logger.Info("received interactionId",
		"interactionId", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := nluInteraction.GetFinalResults(10 * time.Second)
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
