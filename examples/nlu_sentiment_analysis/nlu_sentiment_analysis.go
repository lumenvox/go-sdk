package main

import (
	lumenvoxSdk "github.com/lumenvox/go-sdk"
	"github.com/lumenvox/go-sdk/config"
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"github.com/lumenvox/go-sdk/session"

	"fmt"
	"log"
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

	// Set audio configuration for session. It's just text processing, so leaving it empty is fine.
	audioConfig := session.AudioConfig{}

	// Create a new session.
	streamTimeout := 5 * time.Minute
	sessionObject, err := client.NewSession(streamTimeout, audioConfig)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err.Error())
	}

	///////////////////////
	// Create NLU interaction
	///////////////////////

	language := "en-US"

	// Configure NLU settings.
	nluSettings := client.GetNluSettings(0, 0, "",
		"", false, false,
		api.NluSettings_UNDEFINED, true, nil)

	inputText := "LumenVox is a company specializing in speech recognition and voice biometrics technology. Their" +
		" solutions are designed to enhance customer interactions by enabling automated speech recognition (ASR) and" +
		" voice-based authentication in various applications, such as call centers and interactive voice response" +
		" (IVR) systems. LumenVox focuses on providing secure, accurate, and scalable voice solutions to streamline" +
		" processes and improve user experiences. Their technology is often used in industries like finance," +
		" healthcare, and telecommunications to provide faster and more secure services by leveraging natural voice" +
		" interactions."

	// Create interaction.
	nluInteraction, err := sessionObject.NewNlu(language, inputText, nluSettings, nil)
	if err != nil {
		log.Printf("failed to create NLU interaction: %v", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := nluInteraction.InteractionId
	log.Printf("received interaction ID: %s", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := nluInteraction.GetFinalResults(10 * time.Second)
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
