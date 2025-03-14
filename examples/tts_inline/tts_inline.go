package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/config"
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"github.com/lumenvox/go-sdk/session"

	"encoding/binary"
	"errors"
	"fmt"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"log"
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
		log.Fatalf("Unable to get config: %v\n", err)
		return
	}

	// Create client and open connection.
	client, err := lumenvoxSdk.CreateClient(
		cfg.ApiEndpoint,
		cfg.EnableTls,
		cfg.CertificatePath,
		cfg.AllowInsecureTls,
		cfg.DeploymentId,
		"", // Auth token unused
	)

	// Catch error from client creation.
	if err != nil {
		log.Fatalf("Failed to create connection: %v\n", err)
		return
	} else {
		log.Printf("Successfully created connection to LumenVox API!")
	}

	///////////////////////
	// Session creation
	///////////////////////

	// Set audio configuration for session. We're not sending any audio to the API,
	// so it's fine to leave it empty.
	audioConfig := session.AudioConfig{}

	// Create a new session.
	streamTimeout := 5 * time.Minute
	sessionObject, err := client.NewSession(streamTimeout, audioConfig)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err.Error())
	}

	///////////////////////
	// Create TTS interaction
	///////////////////////

	language := "en-US"
	voiceName := "Chris"
	textToSynthesize := "Hello World!"

	inlineSettings := client.GetTtsInlineSynthesisSettings(voiceName, "", "", "", "", "")

	// Note: conversion to WAV format happens after we get the audio back
	audioSampleRate := int32(16000)
	synthesizedAudioFormat := &api.AudioFormat{
		StandardAudioFormat: api.AudioFormat_STANDARD_AUDIO_FORMAT_LINEAR16,
		SampleRateHertz:     &api.OptionalInt32{Value: audioSampleRate},
	}

	// Create interaction.
	ttsInteraction, err := sessionObject.NewInlineTts(language, textToSynthesize, inlineSettings, synthesizedAudioFormat, nil, nil)
	if err != nil {
		log.Printf("failed to create interaction: %v", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := ttsInteraction.InteractionId
	log.Printf("received interaction ID: %s", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := ttsInteraction.GetFinalResults(10 * time.Second)
	if err != nil {
		fmt.Printf("error while waiting for final results: %v\n", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	} else {
		fmt.Printf("got final results: %v\n", finalResults)
	}

	///////////////////////
	// Get TTS audio data
	///////////////////////

	// Create the output directory if it doesn't already exist.
	audioOutputFolder := "./examples/synthesized_audio/"
	_ = os.MkdirAll(audioOutputFolder, os.ModePerm)

	synthesisFilename := audioOutputFolder + "inline-Chris.wav"

	// Pull the audio from the synthesis.
	synthesizedAudioData, err := sessionObject.PullTtsAudio(interactionId, 0, 0, 0)
	if err != nil {
		fmt.Printf("error pulling audio: %v\n", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages.
		return
	}

	// Convert the data to WAV.
	var intData []int
	intData = twoByteDataToIntSlice(synthesizedAudioData)
	err = saveWavFile(synthesisFilename, audioSampleRate, intData)
	if err != nil {
		log.Printf("Failed to save audio: %v", err.Error())
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages.
		return
	}

	log.Printf("TTS file [%s] generated with bytes: [%d]", synthesisFilename, len(synthesizedAudioData))

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
