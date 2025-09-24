package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/auth"
	"github.com/lumenvox/go-sdk/config"
	"github.com/lumenvox/go-sdk/logging"
	"github.com/lumenvox/go-sdk/session"

	"github.com/lumenvox/protos-go/lumenvox/api"

	"encoding/binary"
	"errors"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
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
	ssmlUrl := "https://assets.lumenvox.com/ssml/SimpleTTSClient.ssml"

	sslVerifyPeer := &api.OptionalBool{Value: false}

	// Note: conversion to WAV format happens after we get the audio back
	audioSampleRate := int32(16000)
	synthesizedAudioFormat := &api.AudioFormat{
		StandardAudioFormat: api.AudioFormat_STANDARD_AUDIO_FORMAT_LINEAR16,
		SampleRateHertz:     &api.OptionalInt32{Value: audioSampleRate},
	}

	// Create interaction.
	var synthesisTimeoutMs *api.OptionalInt32 = nil
	var generalInteractionSettings *api.GeneralInteractionSettings = nil
	var enablePartialResults *api.OptionalBool = nil
	ttsInteraction, err := sessionObject.NewUrlTts(language, ssmlUrl, sslVerifyPeer,
		synthesizedAudioFormat, synthesisTimeoutMs, generalInteractionSettings, enablePartialResults)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := ttsInteraction.InteractionId
	logger.Info("received interactionId",
		"interactionId", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := ttsInteraction.GetFinalResults(10 * time.Second)
	if err != nil {
		logger.Error("waiting for final results",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	} else {
		logger.Info("got final results",
			"finalResults", finalResults)
	}

	///////////////////////
	// Get TTS audio data
	///////////////////////

	// Create the output directory if it doesn't already exist.
	audioOutputFolder := "./examples/synthesized_audio/"
	_ = os.MkdirAll(audioOutputFolder, os.ModePerm)

	synthesisFilename := audioOutputFolder + "uri-Chris.wav"

	// Pull the audio from the synthesis.
	synthesizedAudioData, err := sessionObject.PullTtsAudio(interactionId, 0, 0, 0)
	if err != nil {
		logger.Error("pulling audio",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages.
		return
	}

	// Convert the data to WAV.
	var intData []int
	intData = twoByteDataToIntSlice(synthesizedAudioData)
	err = saveWavFile(synthesisFilename, audioSampleRate, intData)
	if err != nil {
		logger.Error("failed to save audio",
			"error", err.Error())
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages.
		return
	}

	logger.Info("TTS file generated",
		"synthesisFilename", synthesisFilename,
		"bytesGenerated", len(synthesizedAudioData))

	///////////////////////
	// Session close
	///////////////////////

	sessionObject.CloseSession()

	// Delay a little to get any residual messages
	time.Sleep(500 * time.Millisecond)

}

// Saves the specified audio to the specified .wav filename using the specified sample rate
func saveWavFile(wavFilename string, audioSampleRate int32, pcmData []int) (err error) {

	// Create output wav file
	var outputWavFile *os.File
	outputWavFile, err = os.Create(wavFilename)
	if err != nil {
		return errors.New("Failed to create .wav file: " + err.Error())
	}
	defer outputWavFile.Close()

	buffer := audio.IntBuffer{
		Format:         &audio.Format{SampleRate: 16000, NumChannels: 1},
		Data:           pcmData,
		SourceBitDepth: 0,
	}

	// Create wav encoder
	wavEncoder := wav.NewEncoder(outputWavFile,
		int(audioSampleRate), //buf.Format.SampleRate,
		int(16),              //wd.BitDepth),
		1,                    // buf.Format.NumChannels,
		1,                    //int(wd.WavAudioFormat)
	)
	defer wavEncoder.Close()

	// Write IntBuffer to output file via encoder
	if err = wavEncoder.Write(&buffer); err != nil {
		return errors.New("Failed to write .wav file: " + err.Error())
	}

	return nil
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
