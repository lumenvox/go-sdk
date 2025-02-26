package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"github.com/lumenvox/go-sdk/session"

	"fmt"
	"log"
	"os"
	"time"
)

// Grammar with lexicon reference for aliases.
var transcriptionGrammar = `<?xml version='1.0'?>
    <grammar xml:lang="en" version="1.0" root="root" mode="voice"
             xmlns="http://www.w3.org/2001/06/grammar"
             tag-format="semantics/1.0.2006">

        <meta name="TRANSCRIPTION_ENGINE" content="V2"/>
        <lexicon uri="https://assets.lumenvox.com/lexicon/zero_lexicon.xml"/>       
        <rule id="root" scope="public">
            <item>

            </item>
        </rule>
    </grammar>`

func main() {

	///////////////////////
	// Client creation
	///////////////////////

	// Set variables for connection to lumenvox API.
	defaultDeploymentId := "00000000-0000-0000-0000-000000000000"

	// Create client and open connection.
	client, err := lumenvoxSdk.CreateClient("localhost:8280", false, "",
		false, defaultDeploymentId, "")

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
		IsBatch:    true,
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
	// "Zero, oh, naught" will result in "Zero Zero Zero" for the final results due to alias/lexicon above.
	audioFilePath := "./examples/test_data/zero_oh_naught.raw"
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
			GrammarLoadMethod: &api.Grammar_InlineGrammarText{
				InlineGrammarText: transcriptionGrammar,
			},
		},
	}

	// Configure VAD settings.
	useVad := false
	bargeInTimeout := int32(30000) // 30 second default
	eosDelay := int32(800)
	vadSettings := client.GetVadSettings(useVad, bargeInTimeout, eosDelay, nil,
		api.VadSettings_NOISE_REDUCTION_MODE_DISABLED, nil, nil, nil, nil, nil)

	// Configure recognition settings.
	decodeTimeout := int32(10000)
	enablePartialResults := false
	recognitionSettings := client.GetRecognitionSettings(decodeTimeout, enablePartialResults, nil, nil, nil)

	// Configure audio consume settings.
	audioConsumeSettings, err := client.GetAudioConsumeSettings(0,
		api.AudioConsumeSettings_AUDIO_CONSUME_MODE_BATCH,
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
