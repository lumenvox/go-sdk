package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/lumenvox/protos-go/lumenvox/api"
)

// NluInteractionObject represents an NLU interaction.
type NluInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.NluInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type nluCreateResult struct {
	InteractionObject *NluInteractionObject
	Error             error
}

type interactionCreateNluHelper struct {
	interactionCreateChannel chan *nluCreateResult
	deadline                 time.Time
}

// prepareInteractionCreateNlu creates a helper in an internal map to prepare
// for an interactionCreateNlu response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateNlu(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *nluCreateResult, err error) {

	session.interactionCreateNluMapLock.Lock()
	defer session.interactionCreateNluMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateNluMap[correlationId]; ok {
		return nil, errors.New("interaction create nlu channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateNluHelper{
		interactionCreateChannel: make(chan *nluCreateResult, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateNluMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewNlu attempts to create a new NLU interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewNlu(language string,
	inputText string,
	nluSettings *api.NluSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *NluInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateNlu(correlationId, interactionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getNluRequest(correlationId, language, inputText,
		nluSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateNluRequest error: %v", err)
		logger.Error("sending NLU create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object or error.
	var result *nluCreateResult
	select {
	case result = <-interactionCreateChan:
	case <-time.After(interactionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "nlu",
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if result.Error != nil {
		// API returned an error, return it to the caller
		return nil, result.Error
	}

	if result.InteractionObject == nil {
		logger.Error("received nil interaction object without error",
			"type", "nlu",
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	interactionObject = result.InteractionObject

	if EnableVerboseLogging {
		logger.Debug("created new NLU interaction",
			"interaction_id", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors.
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures do not trigger errors from this function, so long as
// the notification arrives before the timeout.
func (nluInteraction *NluInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-nluInteraction.resultsReadyChannel:
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
func (nluInteraction *NluInteractionObject) GetFinalResults(timeout time.Duration) (result *api.NluInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = nluInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if nluInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if nluInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_NLU_RESULT {
			// Successful interaction, return result
			return nluInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", nluInteraction.FinalResultStatus, nluInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of NLU interaction")
	}
}
