package main

import (
	"fmt"

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
	cfg, err := config.GetConfigValues("../../config_values.ini")
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
	audioFilePath := "../../examples/test_data/call_screening.ulaw"
	audioData, err = os.ReadFile(audioFilePath)
	if err != nil {
		logger.Error("reading audio file",
			"error", err.Error())
		sessionObject.CloseSession()
		os.Exit(1)
	}

	// Queue the audio for the internal streamer.
	err = sessionObject.AddAudio(audioData)
	if err != nil {
		logger.Error("failed to queue audio",
			"error", err.Error())
		sessionObject.CloseSession()
		os.Exit(1)
	}

	language := "en-US"

	// CPA Grammar
	cpaGrammar := []*api.Grammar{
		{
			GrammarLoadMethod: &api.Grammar_GrammarUrl{
				GrammarUrl: "http://file-server/Grammar/cpa/default_cpa.grxml",
			},
			Label: &api.OptionalString{
				Value: "cpa-grammar",
			},
		},
	}

	// CPA prompt end grammar
	promptEndGrammar := []*api.Grammar{
		{
			GrammarLoadMethod: &api.Grammar_GrammarUrl{
				GrammarUrl: "http://file-server/Grammar/call_screening/cpa_prompt_end.grxml",
			},
			Label: &api.OptionalString{
				Value: "prompt-end-grammar",
			},
		},
	}

	// AMD detect screening tone grammar
	amdScreeningToneGrammar := []*api.Grammar{
		{
			GrammarLoadMethod: &api.Grammar_GrammarUrl{
				GrammarUrl: "http://file-server/Grammar/call_screening/amd_screening_tone.grxml",
			},
			Label: &api.OptionalString{
				Value: "screening-tone-detect-grammar",
			},
		},
	}

	// CPA silence timeout grammar
	cpaSilenceTimeoutGrammar := []*api.Grammar{
		{
			GrammarLoadMethod: &api.Grammar_GrammarUrl{
				GrammarUrl: "http://file-server/Grammar/call_screening/cpa_unknown_silence_timeout.grxml",
			},
			Label: &api.OptionalString{
				Value: "call-screening-tone-start-grammar",
			},
		},
	}

	///////////////////////
	// Create Grammar Based CPA Interaction
	///////////////////////

	var grammarSettings *api.GrammarSettings = nil

	// Configure VAD settings.
	const useVad = true
	bargeInTimeout := int32(30000) // 30-second default
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
	const enablePartialResults = false
	var maxAlternatives *api.OptionalInt32 = nil
	var trimSilence *api.OptionalInt32 = nil
	var confidenceThreshold *api.OptionalInt32 = nil
	recognitionSettings := client.GetRecognitionSettings(decodeTimeout, enablePartialResults,
		maxAlternatives, trimSilence, confidenceThreshold)

	// Configure audio consume settings.
	var audioChannel int32 = 0
	audioConsumeMode := api.AudioConsumeSettings_AUDIO_CONSUME_MODE_STREAMING
	streamStartLocation := api.AudioConsumeSettings_STREAM_START_LOCATION_INTERACTION_CREATED
	var startOffsetMs *api.OptionalInt32 = nil
	var audioConsumeMaxMs *api.OptionalInt32 = nil
	audioConsumeSettings, err := client.GetAudioConsumeSettings(audioChannel,
		audioConsumeMode, streamStartLocation, startOffsetMs, audioConsumeMaxMs)

	var generalInteractionSettings *api.GeneralInteractionSettings = nil

	// Create interaction.
	asrInteraction, err := sessionObject.NewAsr(language, cpaGrammar, grammarSettings, recognitionSettings,
		vadSettings, audioConsumeSettings, generalInteractionSettings)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := asrInteraction.InteractionId
	logger.Info("received interaction_id",
		"interaction_id", interactionId)

	// Now that we have waited for the end of speech, wait for the final results to become available.
	finalResults, err := asrInteraction.GetFinalResults(10 * time.Second)
	if err != nil {
		logger.Error("waiting for final results",
			"error", err)
	} else {
		logger.Info("got final results",
			"final_results", finalResults)
	}

	///////////////////////
	// Wait for speech stopped
	///////////////////////

	logger.Info("sending request for speech stopped")

	// Create interaction.
	asrInteraction, err = sessionObject.NewAsr(language, promptEndGrammar, grammarSettings, recognitionSettings,
		vadSettings, audioConsumeSettings, generalInteractionSettings)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId = asrInteraction.InteractionId
	logger.Info("received interaction_id",
		"interaction_id", interactionId)

	// Now that we have waited for the end of speech, wait for the final results to become available.
	finalResults, err = asrInteraction.GetFinalResults(10 * time.Second)
	if err != nil {
		logger.Error("waiting for final results",
			"error", err)
	} else {
		logger.Info("got final results",
			"final_results", finalResults)
	}

	///////////////////////
	// Wait for warble tone start
	///////////////////////

	fmt.Println("")
	logger.Info("sending request for call screening tone start")

	// Create interaction.
	asrInteraction, err = sessionObject.NewAsr(language, amdScreeningToneGrammar, grammarSettings, recognitionSettings,
		vadSettings, audioConsumeSettings, generalInteractionSettings)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId = asrInteraction.InteractionId
	logger.Info("received interaction_id",
		"interaction_id", interactionId)

	// Now that we have waited for the end of speech, wait for the final results to become available.
	finalResults, err = asrInteraction.GetFinalResults(120 * time.Second)
	if err != nil {
		logger.Error("waiting for final results",
			"error", err)
	} else {
		logger.Info("got final results",
			"final_results", finalResults)
	}

	///////////////////////
	// CPA interaction and ignore call screening tone
	///////////////////////

	fmt.Println("")
	logger.Info("sending request for call screening tone ignore")

	// Create interaction.
	asrInteraction, err = sessionObject.NewAsr(language, cpaSilenceTimeoutGrammar, grammarSettings, recognitionSettings,
		vadSettings, audioConsumeSettings, generalInteractionSettings)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId = asrInteraction.InteractionId
	logger.Info("received interaction_id",
		"interaction_id", interactionId)

	// Now that we have waited for the end of speech, wait for the final results to become available.
	finalResults, err = asrInteraction.GetFinalResults(120 * time.Second)
	if err != nil {
		logger.Error("waiting for final results",
			"error", err)
	} else {
		logger.Info("got final results",
			"final_results", finalResults)
	}

	///////////////////////
	// Session close
	///////////////////////

	sessionObject.CloseSession()
}
