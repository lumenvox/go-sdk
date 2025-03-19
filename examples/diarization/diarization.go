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

	// Set audio configuration for session.
	audioConfig := session.AudioConfig{
		Format:     api.AudioFormat_STANDARD_AUDIO_FORMAT_ULAW,
		SampleRate: 8000,
		IsBatch:    true,
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
	// Add audio to session
	///////////////////////

	var audioData []byte

	// Read data from disk.
	audioFilePath := "./examples/test_data/Capacity-AI-Powered-Support-Automation-Platform-English-1min.ulaw"
	audioData, err = os.ReadFile(audioFilePath)
	if err != nil {
		logger.Error("reading audio file",
			"error", err.Error())
		os.Exit(1)
	}

	// Queue the audio for the internal streamer.
	sessionObject.AddAudio(audioData)

	///////////////////////
	// Create diarization interaction
	///////////////////////

	language := "en-US"

	// Configure audio consume settings.
	audioConsumeSettings, err := client.GetAudioConsumeSettings(0,
		api.AudioConsumeSettings_AUDIO_CONSUME_MODE_BATCH,
		api.AudioConsumeSettings_STREAM_START_LOCATION_STREAM_BEGIN,
		nil,
		nil,
	)

	// Create interaction.
	diarizationInteraction, err := sessionObject.NewDiarization(language, 2, nil,
		nil, audioConsumeSettings)

	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := diarizationInteraction.InteractionId
	logger.Info("received interactionId",
		"interactionId", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := diarizationInteraction.GetFinalResults(10 * time.Second)
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
