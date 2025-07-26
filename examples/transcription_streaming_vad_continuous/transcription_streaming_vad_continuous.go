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
	audioFilePath := "./examples/test_data/the_great_gatsby_1_minute.ulaw"
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
	bargeInTimeout := int32(-1) // unlimited
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
	decodeTimeout := int32(30000)
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

	// to enable continuous transcription:
	enableContinuousTranscription := &api.OptionalBool{Value: true}

	// Other transcription settings
	var phrases []*api.TranscriptionPhraseList = nil
	var embeddedGrammars []*api.Grammar = nil
	var normalizationSettings *api.NormalizationSettings = nil
	languageModelName := ""
	acousticModelName := ""
	enablePostProcessing := ""

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

	// Set up a goroutine to finalize the interaction once we have finished
	// streaming all the audio. In a production environment, you will probably
	// want some other mechanism to determine when you are done processing
	// speech.
	go func() {
		// For this example, we'll wait for the audio to finish streaming before calling
		// finalize. This is not a standard use case. Typically, you'd use batch for
		// something like this.
		for sessionObject.AudioBufferSize() > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		// The audio has finished streaming. Finalize the interaction.
		err = sessionObject.FinalizeInteraction(interactionId)
		if err != nil {
			logger.Error("failed to finalize interaction",
				"error", err.Error())
		} else {
			logger.Debug("finalized interaction")
		}
	}()

	err = transcriptionInteraction.WaitForBeginProcessing(0, 10*time.Second)
	if err != nil {
		logger.Error("waiting for begin processing",
			"error", err)
		sessionObject.CloseSession()
		return
	}
	logger.Debug("got begin processing")

	err = transcriptionInteraction.WaitForBargeIn(0, 10*time.Second)
	if err != nil {
		logger.Error("waiting for barge in",
			"error", err)
		sessionObject.CloseSession()
		return
	}
	logger.Debug("got barge in")

	// set up loop to listen for any results, partial or otherwise
	finalResultReceived := false
	resultIdx := 0
	for finalResultReceived == false {

		// if we haven't finalized, wait for the next result
		resultIdx, finalResultReceived, err = transcriptionInteraction.WaitForNextResult(20 * time.Second)
		if err != nil {
			logger.Error("waiting for results",
				"error", err)
		} else if finalResultReceived == false {
			partialResult, err := transcriptionInteraction.GetPartialResult(resultIdx)
			if err != nil {
				logger.Error("getting partial result",
					"error", err)
			} else {
				// the transcriptionResult here has lots of fields. For the example, we're only looking
				// at the transcript.
				transcriptionResult := partialResult.PartialResult.GetTranscriptionInteractionResult()
				if len(transcriptionResult.NBests) > 0 {
					transcript := transcriptionResult.NBests[0].AsrResultMetaData.Transcript
					logger.Info("partial result",
						"resultIdx", resultIdx,
						"transcript", transcript)
				}
			}
		} else {
			// Continuous transcription does not produce final results in a standard way. It will send
			// a final results message, which can be used to detect the end of the interaction, but that
			// message will not contain any actual results.
			logger.Info("interaction complete")
			if transcriptionInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_ERROR {
				logger.Error("interaction ended with error",
					"status", transcriptionInteraction.FinalStatus)
			}
		}
	}

	///////////////////////
	// Session close
	///////////////////////

	sessionObject.CloseSession()

}
