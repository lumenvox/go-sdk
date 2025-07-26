package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
)

// AmdInteractionObject represents an AMD interaction.
type AmdInteractionObject struct {
	InteractionId string

	// VAD begin processing tracking
	vadBeginProcessingReceived bool
	vadBeginProcessingChannel  chan struct{}

	// VAD barge-in tracking
	vadBargeInReceived int
	vadBargeInChannel  chan int

	// VAD barge-out tracking
	vadBargeOutReceived int
	vadBargeOutChannel  chan int

	// VAD barge-in timeout tracking
	vadBargeInTimeoutReceived bool
	vadBargeInTimeoutChannel  chan struct{}

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.AmdInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateAmdHelper struct {
	interactionCreateChannel chan *AmdInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateAmd creates a helper in an internal map to prepare
// for an interactionCreateAmd response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateAmd(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *AmdInteractionObject, err error) {

	session.interactionCreateAmdMapLock.Lock()
	defer session.interactionCreateAmdMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateAmdMap[correlationId]; ok {
		return nil, errors.New("interaction create amd channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateAmdHelper{
		interactionCreateChannel: make(chan *AmdInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateAmdMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewAmd attempts to create a new AMD interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewAmd(
	amdSettings *api.AmdSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	vadSettings *api.VadSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *AmdInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a channel to wait for the response
	interactionCreateChan, err := session.prepareInteractionCreateAmd(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getAmdRequest(correlationId,
		amdSettings, audioConsumeSettings, vadSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateAmdRequest error: %v", err)
		logger.Error("sending AMD create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "amd",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "amd",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new AMD interaction",
			"interactionId", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// WaitForBeginProcessing blocks until the API sends a VAD_BEGIN_PROCESSING
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// an error will be returned.
func (amdInteraction *AmdInteractionObject) WaitForBeginProcessing(timeout time.Duration) error {

	if amdInteraction.vadBeginProcessingReceived {
		return nil
	}

	select {
	case <-amdInteraction.vadBeginProcessingChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeIn blocks until the API sends a VAD_BARGE_IN message. If the
// message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, an error will be
// returned.
func (amdInteraction *AmdInteractionObject) WaitForBargeIn(timeout time.Duration) error {

	if amdInteraction.vadBargeInReceived != -1 {
		return nil
	}

	select {
	case <-amdInteraction.vadBargeInChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForEndOfSpeech blocks until the API sends a VAD_END_OF_SPEECH message. If
// the message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, an error will be returned.
func (amdInteraction *AmdInteractionObject) WaitForEndOfSpeech(timeout time.Duration) error {

	if amdInteraction.vadBargeOutReceived != -1 {
		return nil
	}

	select {
	case <-amdInteraction.vadBargeOutChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeInTimeout blocks until the API sends a VAD_BARGE_IN_TIMEOUT
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// an error will be returned.
func (amdInteraction *AmdInteractionObject) WaitForBargeInTimeout(timeout time.Duration) error {

	if amdInteraction.vadBargeInTimeoutReceived {
		return nil
	}

	// vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
	select {
	case <-amdInteraction.vadBargeInTimeoutChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors (like a
// barge-in timeout).
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures (like a barge-in timeout) do not trigger errors from
// this function, so long as the notification arrives before the timeout.
func (amdInteraction *AmdInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-amdInteraction.resultsReadyChannel:
		// resultsReadyChannel is closed when final results arrive
		return nil
	case <-amdInteraction.vadBargeInTimeoutChannel:
		// vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
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
func (amdInteraction *AmdInteractionObject) GetFinalResults(timeout time.Duration) (result *api.AmdInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = amdInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if amdInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if amdInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_AMD_TONE {
			// Successful interaction, return result
			return amdInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", amdInteraction.FinalResultStatus, amdInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else if amdInteraction.vadBargeInTimeoutReceived {
		// If we received a barge-in timeout, return an error.
		return nil, errors.New("barge-in timeout")
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of AMD interaction")
	}
}
