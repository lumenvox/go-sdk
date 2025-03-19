package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
	"fmt"
	"time"
)

// DiarizationInteractionObject represents a diarization interaction.
type DiarizationInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.DiarizationInteractionResult
	resultsReadyChannel  chan struct{}
}

// NewDiarization attempts to create a new diarization interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewDiarization(
	language string,
	maxSpeakers int32,
	requestTimeoutMs *api.OptionalInt32,
	generalInteractionSettings *api.GeneralInteractionSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
) (interactionObject *DiarizationInteractionObject, err error) {

	logger := getLogger()

	// Create diarization interaction, adding parameters such as VAD and recognition settings

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getDiarizationRequest("", language, maxSpeakers,
		requestTimeoutMs, generalInteractionSettings, audioConsumeSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateDiarizationRequest error: %v", err)
		logger.Error("sending diarization create request",
			"error", err.Error())
		return nil, err
	}

	// Get the interaction ID.
	diarizationResponse := <-session.createdDiarizationChannel
	interactionId := diarizationResponse.InteractionId
	if EnableVerboseLogging {
		logger.Debug("created new diarization interaction",
			"interactionId", interactionId)
	}

	// Create the interaction object.
	interactionObject = &DiarizationInteractionObject{
		InteractionId:        interactionId,
		finalResultsReceived: false,
		finalResults:         nil,
		resultsReadyChannel:  make(chan struct{}),
	}

	// Add the interaction object to the session
	session.diarizationInteractionsMap[interactionId] = interactionObject

	return interactionObject, err
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors (like a
// barge-in timeout).
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures (like a barge-in timeout) do not trigger errors from
// this function, so long as the notification arrives before the timeout.
func (diarizationInteraction *DiarizationInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-diarizationInteraction.resultsReadyChannel:
		// resultsReadyChannel is closed when final results arrive
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, like when a barge-in timeout is received, an error
// describing the issue will be returned.
//
// If the interaction does not end before the specified timeout, an error will
// be returned.
func (diarizationInteraction *DiarizationInteractionObject) GetFinalResults(timeout time.Duration) (*api.DiarizationInteractionResult, error) {

	// Wait for the end of the interaction.
	err := diarizationInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if diarizationInteraction.finalResultsReceived {
		// If we received final results, return them.
		return diarizationInteraction.finalResults, nil
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of diarization interaction")
	}
}
