package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/lumenvox/protos-go/lumenvox/api"
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

type diarizationCreateResult struct {
	InteractionObject *DiarizationInteractionObject
	Error             error
}

type interactionCreateDiarizationHelper struct {
	interactionCreateChannel chan *diarizationCreateResult
	deadline                 time.Time
}

// prepareInteractionCreateDiarization creates a helper in an internal map to prepare
// for an interactionCreateDiarization response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateDiarization(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *diarizationCreateResult, err error) {

	session.interactionCreateDiarizationMapLock.Lock()
	defer session.interactionCreateDiarizationMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateDiarizationMap[correlationId]; ok {
		return nil, errors.New("interaction create diarization channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateDiarizationHelper{
		interactionCreateChannel: make(chan *diarizationCreateResult, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateDiarizationMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
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

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateDiarization(correlationId, interactionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getDiarizationRequest(correlationId, language, maxSpeakers,
		requestTimeoutMs, generalInteractionSettings, audioConsumeSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateDiarizationRequest error: %v", err)
		logger.Error("sending diarization create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object or error.
	var result *diarizationCreateResult
	select {
	case result = <-interactionCreateChan:
	case <-time.After(interactionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "diarization",
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
			"type", "diarization",
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	interactionObject = result.InteractionObject

	if EnableVerboseLogging {
		logger.Debug("created new diarization interaction",
			"interaction_id", interactionObject.InteractionId)
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
