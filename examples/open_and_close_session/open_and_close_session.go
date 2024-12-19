package main

import (
    lumenvoxSdk "github.com/lumenvox/go-sdk"
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "github.com/lumenvox/go-sdk/session"
    "fmt"
    "log"
    "time"
)

func main() {

    fmt.Println("")

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
        Format:     api.AudioFormat_STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE,
        SampleRate: 0,
        IsBatch:    false,
    }

    // Create a new session. Note: we have specified NO_AUDIO_RESOURCE because
    // we're not sending any audio to the API.
    streamTimeout := 5 * time.Minute
    sessionObject, err := client.NewSession(streamTimeout, audioConfig)
    if err != nil {
        log.Fatalf("Failed to create session: %v", err.Error())
    }

    ///////////////////////
    // Session close
    ///////////////////////

    sessionObject.CloseSession()

    // Delay a little to get any residual messages
    time.Sleep(500 * time.Millisecond)
}
