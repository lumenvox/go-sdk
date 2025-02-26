package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/session"

	"fmt"
	"log"
	"time"
)

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

	// Set audio configuration for session. It's just text processing, so leaving it empty is fine.
	audioConfig := session.AudioConfig{}

	// Create a new session.
	streamTimeout := 5 * time.Minute
	sessionObject, err := client.NewSession(streamTimeout, audioConfig)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err.Error())
	}

	///////////////////////
	// Create normalization interaction
	///////////////////////

	language := "en-US"

	// Configure normalization settings.
	enableCapitalization := true
	normalizationSettings := client.GetNormalizationSettings(false, enableCapitalization, false, false, false, nil)

	textToNormalize := "recorded books presents an unabridged recording of the great gatsby by f scott" +
		" fitzgerald narrated by frank muller chapter one in my younger and more vulnerable years my father" +
		" gave me some advice that i've been turning over in my mind ever since whenever you feel like criticizing" +
		" anyone he told me just remember that all the people in this world haven't had the advantages that you've" +
		" had he didn't say any more but we've always been unusually communicative and a reserved way and i" +
		" understood that he meant a great deal more than that in consequence i am inclined to reserve all judgments" +
		" the habit that has opened up many curious natures to me and also made me the victim of not a few veteran" +
		" wars the abnormal mind is quick to detect and detach itself to this quality when it appears in a normal person"

	// Create interaction.
	normalizationInteraction, err := sessionObject.NewNormalization(language, textToNormalize,
		normalizationSettings, nil)
	if err != nil {
		log.Printf("failed to create interaction: %v", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := normalizationInteraction.InteractionId
	log.Printf("received interaction ID: %s", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := normalizationInteraction.GetFinalResults(10 * time.Second)
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
