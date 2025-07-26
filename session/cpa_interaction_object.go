package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
)

// CpaInteractionObject represents a CPA interaction.
type CpaInteractionObject struct {
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
	finalResults         *api.CpaInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateCpaHelper struct {
	interactionCreateChannel chan *CpaInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateCpa creates a helper in an internal map to prepare
// for an interactionCreateCpa response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateCpa(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *CpaInteractionObject, err error) {

	session.interactionCreateCpaMapLock.Lock()
	defer session.interactionCreateCpaMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateCpaMap[correlationId]; ok {
		return nil, errors.New("interaction create cpa channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateCpaHelper{
		interactionCreateChannel: make(chan *CpaInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateCpaMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewCpa attempts to create a new CPA interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewCpa(
	cpaSettings *api.CpaSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	vadSettings *api.VadSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *CpaInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateCpa(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getCpaRequest(correlationId,
		cpaSettings, audioConsumeSettings, vadSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateCpaRequest error: %v", err)
		logger.Error("sending CPA create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "cpa",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "cpa",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new CPA interaction",
			"interactionId", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// WaitForBeginProcessing blocks until the API sends a VAD_BEGIN_PROCESSING
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// an error will be returned.
func (cpaInteraction *CpaInteractionObject) WaitForBeginProcessing(timeout time.Duration) error {

	if cpaInteraction.vadBeginProcessingReceived {
		return nil
	}

	select {
	case <-cpaInteraction.vadBeginProcessingChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeIn blocks until the API sends a VAD_BARGE_IN message. If the
// message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, an error will be
// returned.
func (cpaInteraction *CpaInteractionObject) WaitForBargeIn(timeout time.Duration) error {

	if cpaInteraction.vadBargeInReceived != -1 {
		return nil
	}

	select {
	case <-cpaInteraction.vadBargeInChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForEndOfSpeech blocks until the API sends a VAD_END_OF_SPEECH message. If
// the message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, an error will be returned.
func (cpaInteraction *CpaInteractionObject) WaitForEndOfSpeech(timeout time.Duration) error {

	if cpaInteraction.vadBargeOutReceived != -1 {
		return nil
	}

	select {
	case <-cpaInteraction.vadBargeOutChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeInTimeout blocks until the API sends a VAD_BARGE_IN_TIMEOUT
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// an error will be returned.
func (cpaInteraction *CpaInteractionObject) WaitForBargeInTimeout(timeout time.Duration) error {

	if cpaInteraction.vadBargeInTimeoutReceived {
		return nil
	}

	// vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
	select {
	case <-cpaInteraction.vadBargeInTimeoutChannel:
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
func (cpaInteraction *CpaInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-cpaInteraction.resultsReadyChannel:
		// resultsReadyChannel is closed when final results arrive
		return nil
	case <-cpaInteraction.vadBargeInTimeoutChannel:
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
func (cpaInteraction *CpaInteractionObject) GetFinalResults(timeout time.Duration) (result *api.CpaInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = cpaInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if cpaInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if cpaInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_CPA_RESULT {
			// Successful interaction, return result
			return cpaInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", cpaInteraction.FinalResultStatus, cpaInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else if cpaInteraction.vadBargeInTimeoutReceived {
		// If we received a barge-in timeout, return an error.
		return nil, errors.New("barge-in timeout")
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of CPA interaction")
	}
}
