package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
)

// DiarizationInteractionObject represents a diarization interaction.
type DiarizationInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.DiarizationInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
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

	// Create diarization interaction, adding parameters such as VAD and recognition settings

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getDiarizationRequest(correlationId, language, maxSpeakers,
		requestTimeoutMs, generalInteractionSettings, audioConsumeSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateDiarizationRequest error: %v", err)
		logger.Error("sending diarization create request",
			"error", err.Error())
		return nil, err
	}

	// Get the interaction ID.
	var response *api.SessionResponse
	select {
	case response = <-interactionCreateChan:
	case <-time.After(20 * time.Second):
		logger.Error("timed out waiting for interaction id",
			"type", "diarization",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction id")
	}
	diarizationResponse := response.GetInteractionCreateDiarization()
	if diarizationResponse == nil {
		logger.Error("received interactionCreate response with unexpected type",
			"expected", "diarization",
			"response", response,
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received interactionCreate response with unexpected type")
	}
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
	{
		session.Lock() // Protect concurrent map access
		defer session.Unlock()

		session.diarizationInteractionsMap[interactionId] = interactionObject
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
func (diarizationInteraction *DiarizationInteractionObject) GetFinalResults(timeout time.Duration) (result *api.DiarizationInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = diarizationInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if diarizationInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if diarizationInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_DIARIZATION_RESULT {
			// Successful interaction, return result
			return diarizationInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", diarizationInteraction.FinalResultStatus, diarizationInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of diarization interaction")
	}
}
