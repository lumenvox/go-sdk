package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
	"fmt"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
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

// NewNlu attempts to create a new NLU interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewNlu(language string,
	inputText string,
	nluSettings *api.NluSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *NluInteractionObject, err error) {

	logger := getLogger()

	// Create NLU interaction, adding specified parameters

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getNluRequest("", language, inputText,
		nluSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateNluRequest error: %v", err)
		logger.Error("sending NLU create request",
			"error", err.Error())
		return nil, err
	}

	// Get the interaction ID.
	nluResponse := <-session.createdNluChannel
	interactionId := nluResponse.InteractionId
	if EnableVerboseLogging {
		logger.Debug("created new NLU interaction",
			"interactionId", interactionId)
	}

	// Create the interaction object.
	interactionObject = &NluInteractionObject{
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

		session.nluInteractionsMap[interactionId] = interactionObject
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
