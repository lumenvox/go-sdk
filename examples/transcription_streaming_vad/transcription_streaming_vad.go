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

	// Set audio configuration for session.
	audioConfig := session.AudioConfig{
		Format:     api.AudioFormat_STANDARD_AUDIO_FORMAT_ULAW,
		SampleRate: 8000,
		IsBatch:    false,
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
	audioFilePath := "./examples/test_data/1234.ulaw"
	audioData, err = os.ReadFile(audioFilePath)
	if err != nil {
		logger.Error("reading audio file",
			"error", err.Error())
		os.Exit(1)
	}

	// Queue the audio for the internal streamer.
	sessionObject.AddAudio(audioData)

	///////////////////////
	// Create transcription interaction
	///////////////////////

	language := "en-US"

	// Configure VAD settings.
	useVad := true
	bargeInTimeout := int32(30000) // 30 second default
	eosDelay := int32(1000)
	var endOfSpeechTimeoutMs *api.OptionalInt32 = nil
	noiseReductionMode := api.VadSettings_NOISE_REDUCTION_MODE_DISABLED
	var bargeInThreshold *api.OptionalInt32 = nil
	var snrSensitivity *api.OptionalInt32 = nil
	var streamInitDelay *api.OptionalInt32 = nil
	var volumeSensitivity *api.OptionalInt32 = nil
	var windBackMs *api.OptionalInt32 = nil
	vadSettings := client.GetVadSettings(useVad, bargeInTimeout, eosDelay, endOfSpeechTimeoutMs,
		noiseReductionMode, bargeInThreshold, snrSensitivity, streamInitDelay, volumeSensitivity, windBackMs)

	// Configure recognition settings.
	decodeTimeout := int32(10000)
	enablePartialResults := false
	var maxAlternatives *api.OptionalInt32 = nil
	var trimSilence *api.OptionalInt32 = nil
	var confidenceThreshold *api.OptionalInt32 = nil
	recognitionSettings := client.GetRecognitionSettings(decodeTimeout, enablePartialResults,
		maxAlternatives, trimSilence, confidenceThreshold)

	// Configure audio consume settings.
	var audioChannel int32 = 0
	audioConsumeMode := api.AudioConsumeSettings_AUDIO_CONSUME_MODE_STREAMING
	streamStartLocation := api.AudioConsumeSettings_STREAM_START_LOCATION_STREAM_BEGIN
	var startOffsetMs *api.OptionalInt32 = nil
	var audioConsumeMaxMs *api.OptionalInt32 = nil
	audioConsumeSettings, err := client.GetAudioConsumeSettings(audioChannel,
		audioConsumeMode, streamStartLocation, startOffsetMs, audioConsumeMaxMs)

	// Other transcription settings
	var phrases []*api.TranscriptionPhraseList = nil
	var embeddedGrammars []*api.Grammar = nil
	var normalizationSettings *api.NormalizationSettings = nil
	languageModelName := ""
	acousticModelName := ""
	enablePostProcessing := ""
	var enableContinuousTranscription *api.OptionalBool = nil

	// Create interaction.
	transcriptionInteraction, err := sessionObject.NewTranscription(language, phrases,
		embeddedGrammars, audioConsumeSettings, normalizationSettings, vadSettings,
		recognitionSettings, languageModelName, acousticModelName, enablePostProcessing,
		enableContinuousTranscription)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := transcriptionInteraction.InteractionId
	logger.Info("received interactionId",
		"interactionId", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the interaction to start processing
	err = transcriptionInteraction.WaitForBeginProcessing(0, 10*time.Second)
	if err != nil {
		logger.Error("waiting for begin processing",
			"error", err)
		sessionObject.CloseSession()
		return
	}
	logger.Debug("got begin processing")

	// Wait for the start of speech
	err = transcriptionInteraction.WaitForBargeIn(0, 10*time.Second)
	if err != nil {
		logger.Error("waiting for barge in",
			"error", err)
		sessionObject.CloseSession()
		return
	}
	logger.Debug("got barge in")

	// Wait for the interaction to finish normally, without calling finalize.
	err = transcriptionInteraction.WaitForEndOfSpeech(0, 10*time.Second)
	if err != nil {
		logger.Error("waiting for end of speech",
			"error", err)
		sessionObject.CloseSession()
		return
	}
	logger.Debug("got end of speech")

	// Now that we have waited for the end of speech, wait for the final results to become available.
	finalResults, err := transcriptionInteraction.GetFinalResults(10 * time.Second)
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
