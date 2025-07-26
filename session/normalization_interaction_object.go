package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
)

// NormalizationInteractionObject represents a normalization interaction.
type NormalizationInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.NormalizeTextResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateNormalizeTextHelper struct {
	interactionCreateChannel chan *NormalizationInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateNormalization creates a helper in an internal map to prepare
// for an interactionCreateNormalization response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateNormalization(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *NormalizationInteractionObject, err error) {

	session.interactionCreateNormalizeTextMapLock.Lock()
	defer session.interactionCreateNormalizeTextMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateNormalizeTextMap[correlationId]; ok {
		return nil, errors.New("interaction create normalization channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateNormalizeTextHelper{
		interactionCreateChannel: make(chan *NormalizationInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateNormalizeTextMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewNormalization attempts to create a new normalization interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewNormalization(language string,
	textToNormalize string,
	normalizationSettings *api.NormalizationSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *NormalizationInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateNormalization(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getNormalizationRequest(correlationId, language, textToNormalize,
		normalizationSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateNormalizationRequest error: %v", err)
		logger.Error("sending normalization create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "itn",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "itn",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new normalization interaction",
			"interactionId", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors.
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures do not trigger errors from this function, so long as
// the notification arrives before the timeout.
func (normalizationInteraction *NormalizationInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-normalizationInteraction.resultsReadyChannel:
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
func (normalizationInteraction *NormalizationInteractionObject) GetFinalResults(timeout time.Duration) (result *api.NormalizeTextResult, err error) {

	// Wait for the end of the interaction.
	err = normalizationInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if normalizationInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if normalizationInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_TEXT_NORMALIZE_RESULT {
			// Successful interaction, return result
			return normalizationInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", normalizationInteraction.FinalResultStatus, normalizationInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of normalization interaction")
	}
}
