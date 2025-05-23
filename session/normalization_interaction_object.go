package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

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

// NewNormalization attempts to create a new ITN normalization interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewNormalization(language string,
	textToNormalize string,
	normalizationSettings *api.NormalizationSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *NormalizationInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a channel to wait for the response
	interactionCreateChan, err := session.prepareInteractionCreate(correlationId)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// Create normalization interaction, adding specified parameters

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

	// Get the interaction ID.
	var response *api.SessionResponse
	select {
	case response = <-interactionCreateChan:
	case <-time.After(20 * time.Second):
		logger.Error("timed out waiting for interaction id",
			"type", "itn",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction id")
	}
	normalizationResponse := response.GetInteractionCreateNormalizeText()
	if normalizationResponse == nil {
		logger.Error("received interactionCreate response with unexpected type",
			"expected", "itn",
			"response", response,
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received interactionCreate response with unexpected type")
	}
	interactionId := normalizationResponse.InteractionId
	if EnableVerboseLogging {
		logger.Debug("created new ITN interaction",
			"interactionId", interactionId)
	}

	// Create the interaction object.
	interactionObject = &NormalizationInteractionObject{
		InteractionId:        interactionId,
		finalResultsReceived: false,
		finalResults:         nil,
		FinalResultStatus:    api.FinalResultStatus_FINAL_RESULT_STATUS_UNSPECIFIED,
		resultsReadyChannel:  make(chan struct{}),
	}

	// Add the interaction object to the session
	{
		session.Lock() // Protect concurrent map access
		defer session.Unlock()

		session.normalizationInteractionsMap[interactionId] = interactionObject
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
