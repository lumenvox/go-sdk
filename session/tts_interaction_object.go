package session

import (
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "errors"
    "fmt"
    "log"
    "time"
)

// TtsInteractionObject represents a TTS interaction.
type TtsInteractionObject struct {
    InteractionId string

    // Final result tracking
    finalResultsReceived bool
    finalResults         *api.TtsInteractionResult
    resultsReadyChannel  chan struct{}
}

// NewInlineTts attempts to create a new inline TTS interaction. If successful,
// a new interaction object will be returned.
func (session *SessionObject) NewInlineTts(language string,
    textToSynthesize string,
    inlineSettings *api.TtsInlineSynthesisSettings,
    synthesizedAudioFormat *api.AudioFormat,
    synthesisTimeoutMs *api.OptionalInt32,
    generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *TtsInteractionObject, err error) {

    // Create TTS interaction, adding specified parameters

    session.streamSendLock.Lock()
    err = session.SessionStream.Send(getInlineTtsRequest("", language, textToSynthesize,
        synthesizedAudioFormat, synthesisTimeoutMs, inlineSettings, generalInteractionSettings))
    session.streamSendLock.Unlock()
    if err != nil {
        session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
        log.Printf("error sending tts create request: %v", err.Error())
        return nil, err
    }

    // Get the interaction ID.
    ttsResponse := <-session.createdTtsChannel
    interactionId := ttsResponse.InteractionId
    if EnableVerboseLogging {
        log.Printf("created new TTS interaction: %s", interactionId)
    }

    // Create the interaction object.
    interactionObject = &TtsInteractionObject{
        interactionId,
        false,
        nil,
        make(chan struct{}),
    }

    // Add the interaction object to the session
    session.ttsInteractionsMap[interactionId] = interactionObject

    return interactionObject, err
}

// NewUrlTts attempts to create a new URL-based TTS interaction. If successful, a new interaction
// object will be returned.
func (session *SessionObject) NewUrlTts(language string,
    ssmlUrl string,
    sslVerifyPeer *api.OptionalBool,
    synthesizedAudioFormat *api.AudioFormat,
    synthesisTimeoutMs *api.OptionalInt32,
    generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *TtsInteractionObject, err error) {

    // Create TTS interaction, adding specified parameters

    session.streamSendLock.Lock()
    err = session.SessionStream.Send(getUrlTtsRequest("", language, ssmlUrl,
        synthesizedAudioFormat, synthesisTimeoutMs, sslVerifyPeer, generalInteractionSettings))
    session.streamSendLock.Unlock()
    if err != nil {
        session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
        log.Printf("error sending tts create request: %v", err.Error())
        return nil, err
    }

    // Get the interaction ID.
    ttsResponse := <-session.createdTtsChannel
    interactionId := ttsResponse.InteractionId
    if EnableVerboseLogging {
        log.Printf("created new TTS interaction: %s", interactionId)
    }

    // Create the interaction object.
    interactionObject = &TtsInteractionObject{
        interactionId,
        false,
        nil,
        make(chan struct{}),
    }

    // Add the interaction object to the session
    session.ttsInteractionsMap[interactionId] = interactionObject

    return interactionObject, err
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors.
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures do not trigger errors from this function, so long as
// the notification arrives before the timeout.
func (ttsInteraction *TtsInteractionObject) WaitForFinalResults(timeout time.Duration) error {

    select {
    case <-ttsInteraction.resultsReadyChannel:
        // resultsReadyChannel is closed when final results arrive
        return nil
    case <-time.After(timeout):
        return TimeoutError
    }
}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, an error describing the issue will be returned.
//
// If the interaction does not end before the specified timeout, an error will
// be returned.
func (ttsInteraction *TtsInteractionObject) GetFinalResults(timeout time.Duration) (*api.TtsInteractionResult, error) {

    // Wait for the end of the interaction.
    err := ttsInteraction.WaitForFinalResults(timeout)
    if err != nil {
        return nil, err
    }

    if ttsInteraction.finalResultsReceived {
        // If we received final results, return them.
        return ttsInteraction.finalResults, nil
    } else {
        // This should never happen.
        return nil, errors.New("unexpected end of tts interaction")
    }
}
