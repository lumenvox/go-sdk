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

func main() {

	fmt.Println("")

	///////////////////////
	// Client creation
	///////////////////////
	//session.EnableVerboseLogging = true

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
	audioFilePath := "./examples/test_data/the_great_gatsby_1_minute.ulaw"
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

	// Configure VAD settings.
	useVad := true
	bargeInTimeout := int32(-1) // unlimited
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

	// to enable continuous transcription:
	enableContinuousTranscription := &api.OptionalBool{Value: true}

	// Create interaction.
	transcriptionInteraction, err := sessionObject.NewTranscription(language, audioConsumeSettings, nil,
		vadSettings, recognitionSettings, "", "", "", enableContinuousTranscription)
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

	// declare a channel to signal to the result handler when we have finalized
	// the interaction. In a production environment, you probably want some
	// other mechanism to determine when to finalize the interaction and when
	// you should stop listening for results. For the purposes of this example,
	// we're just waiting for our outbound audio stream to complete.
	finalizeChannel := make(chan struct{})
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
			log.Printf("failed to finalize interaction: %v", err.Error())
		} else {
			fmt.Println("finalized interaction")
		}

		close(finalizeChannel)
	}()

	err = transcriptionInteraction.WaitForBeginProcessing(0, 10*time.Second)
	if err != nil {
		fmt.Printf("error while waiting for begin processing: %+v\n", err)
		sessionObject.CloseSession()
		return
	}
	fmt.Println("got begin processing")

	err = transcriptionInteraction.WaitForBargeIn(0, 10*time.Second)
	if err != nil {
		fmt.Printf("error while waiting for barge in: %+v\n", err)
		sessionObject.CloseSession()
		return
	}
	fmt.Println("got barge in")

	// set up loop to listen for any results, partial or otherwise
	finalResultReceived := false
	resultIdx := 0
	for finalResultReceived == false {

		// detect if we have finalized the interaction
		finalizedInteraction := false
		select {
		case <-finalizeChannel:
			finalizedInteraction = true
		default:
		}
		if finalizedInteraction {
			break
		}

		// if we haven't finalized, wait for the next result
		resultIdx, finalResultReceived, err = transcriptionInteraction.WaitForNextResult(20 * time.Second)
		if err != nil {
			fmt.Printf("error while waiting for results: %+v\n", err)
		} else if finalResultReceived == false {
			partialResult, err := transcriptionInteraction.GetPartialResult(resultIdx)
			if err != nil {
				fmt.Printf("Error getting partial result: %+v", err)
			} else {
				// the transcriptionResult here has lots of fields. For the example, we're only looking
				// at the transcript.
				transcriptionResult := partialResult.PartialResult.GetTranscriptionInteractionResult()
				transcript := transcriptionResult.NBests[0].AsrResultMetaData.Transcript
				fmt.Printf("PARTIAL RESULT %d: %+v\n", resultIdx, transcript)
			}
		} else {
			// Continuous transcription does not produce final results in a standard way. It will send
			// a final results message, which can be used to detect the end of the interaction, but that
			// message will not contain any actual results.
			fmt.Println("Interaction complete")
		}
	}

	///////////////////////
	// Session close
	///////////////////////

	sessionObject.CloseSession()

}
