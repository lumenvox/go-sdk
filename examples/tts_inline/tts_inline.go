package main

import (
	"encoding/binary"
	"errors"
	"os"
	"time"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"github.com/lumenvox/protos-go/lumenvox/api"

	lumenvoxSdk "github.com/lumenvox/go-sdk/v2"
	"github.com/lumenvox/go-sdk/v2/auth"
	"github.com/lumenvox/go-sdk/v2/config"
	"github.com/lumenvox/go-sdk/v2/logging"
	"github.com/lumenvox/go-sdk/v2/session"
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

	// Set audio configuration for session. We're not sending any audio to the API,
	// so the format should be set to NO_AUDIO_RESOURCE.
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
	// Create TTS interaction
	///////////////////////

	language := "en-US"
	voiceName := "Chris"
	textToSynthesize := "Hello World!"

	// Configure settings for inline TTS
	voiceGender := ""
	voiceEmphasis := ""
	voicePitch := ""
	voiceRate := ""
	voiceVolume := ""
	inlineSettings := client.GetTtsInlineSynthesisSettings(voiceName, voiceGender,
		voiceEmphasis, voicePitch, voiceRate, voiceVolume)

	// Note: conversion to WAV format happens after we get the audio back
	audioSampleRate := int32(16000)
	synthesizedAudioFormat := &api.AudioFormat{
		StandardAudioFormat: api.AudioFormat_STANDARD_AUDIO_FORMAT_LINEAR16,
		SampleRateHertz:     &api.OptionalInt32{Value: audioSampleRate},
	}

	// Create interaction.
	var synthesisTimeoutMs *api.OptionalInt32 = &api.OptionalInt32{Value: 60000}
	var generalInteractionSettings *api.GeneralInteractionSettings = nil
	var enablePartialResults *api.OptionalBool = nil
	ttsInteraction, err := sessionObject.NewInlineTts(language, textToSynthesize, inlineSettings,
		synthesizedAudioFormat, synthesisTimeoutMs, generalInteractionSettings, enablePartialResults,
		api.InteractionCreateTtsRequest_TTS_ENGINE_UNSPECIFIED)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	} else {
		logger.Info("created interaction")
	}

	interactionId := ttsInteraction.InteractionId
	logger.Info("received interaction_id",
		"interaction_id", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := ttsInteraction.GetFinalResults(60 * time.Second) // should be close to synthesisTimeoutMs
	if err != nil {
		logger.Error("waiting for final results",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	} else {
		logger.Info("got final results",
			"audio_format", finalResults.AudioFormat.String(),
			"sample_rate", finalResults.AudioFormat.SampleRateHertz.Value)
	}

	///////////////////////
	// Get TTS audio data
	///////////////////////

	// Create the output directory if it doesn't already exist.
	audioOutputFolder := "../../examples/synthesized_audio/"
	_ = os.MkdirAll(audioOutputFolder, os.ModePerm)

	synthesisFilename := audioOutputFolder + "inline-" + voiceName + "-sample.wav"

	// Pull the audio from the synthesis.
	synthesizedAudioData, err := sessionObject.PullTtsAudio(interactionId, 0, 0, 0)
	if err != nil {
		logger.Error("pulling audio",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages.
		return
	}

	err = writeAudioToWav(synthesisFilename, synthesizedAudioData, int(audioSampleRate), 1,
		synthesizedAudioFormat.StandardAudioFormat)
	if err != nil {
		logger.Error("failed to save audio",
			"error", err.Error())
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages.
		return
	}

	logger.Info("TTS file generated",
		"synthesis_filename", synthesisFilename,
		"bytes_generated", len(synthesizedAudioData))

	///////////////////////
	// Session close
	///////////////////////

	sessionObject.CloseSession()

	// Delay a little to get any residual messages
	time.Sleep(500 * time.Millisecond)

}

// Writes audio data to a WAV file in the specified format.
func writeAudioToWav(wavFilename string, audioData []byte, sampleRate int, numChannels int,
	audioFormat api.AudioFormat_StandardAudioFormat) (err error) {

	// Create output wav file
	var outputWavFile *os.File
	outputWavFile, err = os.Create(wavFilename)
	if err != nil {
		return errors.New("Failed to create .wav file: " + err.Error())
	}
	defer outputWavFile.Close()

	var (
		audioFormatCode int
		bitDepth        int
		buf             *audio.IntBuffer
	)

	switch audioFormat {
	case api.AudioFormat_STANDARD_AUDIO_FORMAT_LINEAR16:
		audioFormatCode = 1 // PCM
		bitDepth = 16       // 16-bit
		intData := twoByteDataToIntSlice(audioData)

		buf = &audio.IntBuffer{
			Format: &audio.Format{
				NumChannels: numChannels,
				SampleRate:  sampleRate,
			},
			SourceBitDepth: bitDepth,
			Data:           intData,
		}

	case api.AudioFormat_STANDARD_AUDIO_FORMAT_ULAW, api.AudioFormat_STANDARD_AUDIO_FORMAT_ALAW:
		bitDepth = 8
		if audioFormat == api.AudioFormat_STANDARD_AUDIO_FORMAT_ULAW {
			audioFormatCode = 7 // μ-law
		} else {
			audioFormatCode = 6 // A-law
		}

		intData := make([]int, len(audioData))
		for i, b := range audioData {
			intData[i] = int(b)
		}

		buf = &audio.IntBuffer{
			Format: &audio.Format{
				NumChannels: numChannels,
				SampleRate:  sampleRate,
			},
			SourceBitDepth: bitDepth,
			Data:           intData,
		}

	default:
		return errors.New("unsupported audio format")
	}

	wavEncoder := wav.NewEncoder(outputWavFile, sampleRate, bitDepth, numChannels, audioFormatCode)

	if err = wavEncoder.Write(buf); err != nil {
		return errors.New("Failed to write .wav file: " + err.Error())
	}

	return wavEncoder.Close()
}

// Convert byte slice into slice of 16-bit integers (for use when writing .wav)
func twoByteDataToIntSlice(audioData []byte) (convertedData []int) {

	convertedData = make([]int, len(audioData)/2)
	for i := 0; i < len(audioData); i += 2 {
		// Convert the pCapturedSamples byte slice to int16 slice for FormatS16 as we go
		value := int(binary.LittleEndian.Uint16(audioData[i : i+2]))
		convertedData[i/2] = value
	}

	return convertedData
}
