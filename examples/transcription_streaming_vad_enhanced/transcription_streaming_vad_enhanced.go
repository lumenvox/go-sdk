package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/config"
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"github.com/lumenvox/go-sdk/session"

	"fmt"
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
		log.Fatalf("Failed to create session: %v", err.Error())
	}

	///////////////////////
	// Add audio to session
	///////////////////////

	var audioData []byte

	// Read data from disk.
	audioFilePath := "./examples/test_data/account_7856_please.ulaw"
	audioData, err = os.ReadFile(audioFilePath)
	if err != nil {
		log.Fatalf("Error reading audio file: %v", err.Error())
	}

	// Queue the audio for the internal streamer.
	sessionObject.AddAudio(audioData)

	///////////////////////
	// Create transcription interaction
	///////////////////////

	language := "en-US"

	// Configure grammar(s).
	grammars := []*api.Grammar{
		{ // Digits grammar
			GrammarLoadMethod: &api.Grammar_GrammarUrl{
				//GrammarUrl: "http://host.docker.internal:9080/Grammar/en-US/en_digits.grxml",
				GrammarUrl: "https://assets.lumenvox.com/grammar/en/en_digits.grxml",
			},
		},
	}

	// Configure VAD settings.
	useVad := true
	bargeInTimeout := int32(30000) // 30 second default
	eosDelay := int32(1000)
	vadSettings := client.GetVadSettings(useVad, bargeInTimeout, eosDelay, nil,
		api.VadSettings_NOISE_REDUCTION_MODE_DISABLED, nil, nil, nil, nil, nil)

	// Configure recognition settings.
	decodeTimeout := int32(10000)
	enablePartialResults := false
	recognitionSettings := client.GetRecognitionSettings(decodeTimeout, enablePartialResults, nil, nil, nil)

	// Configure audio consume settings.
	audioConsumeSettings, err := client.GetAudioConsumeSettings(0,
		api.AudioConsumeSettings_AUDIO_CONSUME_MODE_STREAMING,
		api.AudioConsumeSettings_STREAM_START_LOCATION_STREAM_BEGIN,
		nil,
		nil,
	)

	// Create interaction.
	transcriptionInteraction, err := sessionObject.NewTranscription(language, grammars, audioConsumeSettings, nil,
		vadSettings, recognitionSettings, "", "", "", nil)
	if err != nil {
		log.Printf("failed to create interaction: %v", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := transcriptionInteraction.InteractionId
	log.Printf("received interaction ID: %s", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the interaction to start processing
	err = transcriptionInteraction.WaitForBeginProcessing(0, 10*time.Second)
	if err != nil {
		fmt.Printf("error while waiting for begin processing: %v\n", err)
		sessionObject.CloseSession()
		return
	}
	fmt.Println("got begin processing")

	// Wait for the start of speech
	err = transcriptionInteraction.WaitForBargeIn(0, 10*time.Second)
	if err != nil {
		fmt.Printf("error while waiting for barge in: %v\n", err)
		sessionObject.CloseSession()
		return
	}
	fmt.Println("got barge in")

	// Wait for the interaction to finish normally, without calling finalize.
	err = transcriptionInteraction.WaitForEndOfSpeech(0, 10*time.Second)
	if err != nil {
		fmt.Printf("error while waiting for end of speech: %v\n", err)
		sessionObject.CloseSession()
		return
	}
	fmt.Println("got end of speech")

	// Now that we have waited for the end of speech, wait for the final results to become available.
	finalResults, err := transcriptionInteraction.GetFinalResults(10 * time.Second)
	if err != nil {
		fmt.Printf("error while waiting for final results: %v\n", err)
	} else {
		fmt.Printf("got final results: %v\n", finalResults)
	}

	///////////////////////
	// Session close
	///////////////////////

	sessionObject.CloseSession()
}
