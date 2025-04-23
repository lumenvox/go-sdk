package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
)

// LanguageIdInteractionObject represents a language id interaction.
type LanguageIdInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.LanguageIdInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

// NewLanguageId attempts to create a new language id interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewLanguageId(
	requestTimeoutMs *api.OptionalInt32,
	generalInteractionSettings *api.GeneralInteractionSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
) (interactionObject *LanguageIdInteractionObject, err error) {

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

	// Create language id interaction, adding parameters such as VAD and recognition settings

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getLanguageIdRequest(correlationId,
		requestTimeoutMs, generalInteractionSettings, audioConsumeSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateLanguageIdRequest error: %v", err)
		logger.Error("sending language id create request",
			"error", err.Error())
		return nil, err
	}

	// Get the interaction ID.
	var response *api.SessionResponse
	select {
	case response = <-interactionCreateChan:
	case <-time.After(20 * time.Second):
		logger.Error("timed out waiting for interaction id",
			"type", "lid",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction id")
	}
	languageIdResponse := response.GetInteractionCreateLanguageId()
	if languageIdResponse == nil {
		logger.Error("received interactionCreate response with unexpected type",
			"expected", "lid",
			"response", response,
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received interactionCreate response with unexpected type")
	}
	interactionId := languageIdResponse.InteractionId
	if EnableVerboseLogging {
		logger.Debug("created new language id interaction",
			"interactionId", interactionId)
	}

	// Create the interaction object.
	interactionObject = &LanguageIdInteractionObject{
		InteractionId:        interactionId,
		finalResultsReceived: false,
		finalResults:         nil,
		resultsReadyChannel:  make(chan struct{}),
	}

	// Add the interaction object to the session
	{
		session.Lock() // Protect concurrent map access
		defer session.Unlock()

		session.languageIdInteractionsMap[interactionId] = interactionObject
	}

	return interactionObject, err
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors (like a
// barge-in timeout).
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures (like a barge-in timeout) do not trigger errors from
// this function, so long as the notification arrives before the timeout.
func (languageIdInteraction *LanguageIdInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-languageIdInteraction.resultsReadyChannel:
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
func (languageIdInteraction *LanguageIdInteractionObject) GetFinalResults(timeout time.Duration) (result *api.LanguageIdInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = languageIdInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if languageIdInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if languageIdInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_LANGUAGE_ID_RESULT {
			// Successful interaction, return result
			return languageIdInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", languageIdInteraction.FinalResultStatus, languageIdInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of language id interaction")
	}
}
