package main

import (
	"os"
	"time"

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

	// Set the audio configuration for the session. It's just text processing, so
	// the format should be set to NO_AUDIO_RESOURCE.
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
	// Create normalization interaction
	///////////////////////

	language := "en-US"

	// Configure normalization settings.
	const enableInverseText = false
	const enableCapitalization = true
	const enableRedaction = false
	const enableSrtGeneration = false
	const enableVttGeneration = false
	var requestTimeoutMs *api.OptionalInt32 = nil
	normalizationSettings := client.GetNormalizationSettings(enableInverseText, enableCapitalization,
		enableRedaction, enableSrtGeneration, enableVttGeneration, requestTimeoutMs)

	textToNormalize := "recorded books presents an unabridged recording of the great gatsby by f scott" +
		" fitzgerald narrated by frank muller chapter one in my younger and more vulnerable years my father" +
		" gave me some advice that i've been turning over in my mind ever since whenever you feel like criticizing" +
		" anyone he told me just remember that all the people in this world haven't had the advantages that you've" +
		" had he didn't say any more but we've always been unusually communicative and a reserved way and i" +
		" understood that he meant a great deal more than that in consequence i am inclined to reserve all judgments" +
		" the habit that has opened up many curious natures to me and also made me the victim of not a few veteran" +
		" wars the abnormal mind is quick to detect and detach itself to this quality when it appears in a normal person"

	// Create interaction.
	var generalInteractionSettings *api.GeneralInteractionSettings = nil
	normalizationInteraction, err := sessionObject.NewNormalization(language, textToNormalize,
		normalizationSettings, generalInteractionSettings)
	if err != nil {
		logger.Error("failed to create interaction",
			"error", err)
		sessionObject.CloseSession()
		time.Sleep(500 * time.Millisecond) // Delay a little to get any residual messages
		return
	}

	interactionId := normalizationInteraction.InteractionId
	logger.Info("received interaction_id",
		"interaction_id", interactionId)

	///////////////////////
	// Get results
	///////////////////////

	// Wait for the final results to arrive.
	finalResults, err := normalizationInteraction.GetFinalResults(10 * time.Second)
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
