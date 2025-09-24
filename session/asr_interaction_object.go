package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"sync"
	"time"
)

// AsrInteractionObject represents an ASR interaction.
type AsrInteractionObject struct {
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

	// VAD barge-out timeout tracking
	vadBargeOutTimeoutReceived bool
	vadBargeOutTimeoutChannel  chan struct{}

	// Partial result tracking
	partialResultLock      sync.Mutex
	partialResultsReceived int
	partialResultsChannels []chan struct{}
	partialResultsList     []*api.PartialResult

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.AsrInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateAsrHelper struct {
	interactionCreateChannel chan *AsrInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateAsr creates a helper in an internal map to prepare
// for an interactionCreateAsr response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateAsr(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *AsrInteractionObject, err error) {

	session.interactionCreateAsrMapLock.Lock()
	defer session.interactionCreateAsrMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateAsrMap[correlationId]; ok {
		return nil, errors.New("interaction create asr channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateAsrHelper{
		interactionCreateChannel: make(chan *AsrInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateAsrMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewAsr attempts to create a new ASR interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewAsr(
	language string,
	grammars []*api.Grammar,
	grammarSettings *api.GrammarSettings,
	recognitionSettings *api.RecognitionSettings,
	vadSettings *api.VadSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *AsrInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateAsr(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getAsrRequest(correlationId, language, grammars, grammarSettings,
		recognitionSettings, vadSettings, audioConsumeSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateAsrRequest error: %v", err)
		logger.Error("sending ASR create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "asr",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "asr",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new ASR interaction",
			"interactionId", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// WaitForBeginProcessing blocks until the API sends a VAD_BEGIN_PROCESSING
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// an error will be returned.
func (asrInteraction *AsrInteractionObject) WaitForBeginProcessing(timeout time.Duration) error {

	if asrInteraction.vadBeginProcessingReceived {
		return nil
	}

	select {
	case <-asrInteraction.vadBeginProcessingChannel:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeIn blocks until the API sends a VAD_BARGE_IN message. If the
// message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, or a barge-in timeout is
// received, an error will be returned.
func (asrInteraction *AsrInteractionObject) WaitForBargeIn(timeout time.Duration) error {

	if asrInteraction.vadBargeInReceived != -1 {
		return nil
	} else if asrInteraction.vadBargeInTimeoutReceived {
		return errors.New("barge-in timeout received")
	}

	select {
	case <-asrInteraction.vadBargeInChannel:
		return nil
	case <-asrInteraction.vadBargeInTimeoutChannel:
		return errors.New("barge-in timeout received")
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForEndOfSpeech blocks until the API sends a VAD_END_OF_SPEECH message. If
// the message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, or an end-of-speech timeout
// arrives, an error will be returned.
func (asrInteraction *AsrInteractionObject) WaitForEndOfSpeech(timeout time.Duration) error {

	if asrInteraction.vadBargeOutReceived != -1 {
		return nil
	} else if asrInteraction.vadBargeOutTimeoutReceived {
		return errors.New("barge-out timeout received")
	}

	select {
	case <-asrInteraction.vadBargeOutChannel:
		return nil
	case <-asrInteraction.vadBargeOutTimeoutChannel:
		return errors.New("barge-out timeout received")
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeInTimeout blocks until the API sends a VAD_BARGE_IN_TIMEOUT
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// or a barge-in arrives, an error will be returned.
func (asrInteraction *AsrInteractionObject) WaitForBargeInTimeout(timeout time.Duration) error {

	if asrInteraction.vadBargeInTimeoutReceived {
		return nil
	} else if asrInteraction.vadBargeInReceived != -1 {
		return errors.New("barge-in received")
	}

	// vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
	select {
	case <-asrInteraction.vadBargeInTimeoutChannel:
		return nil
	case <-asrInteraction.vadBargeInChannel:
		return errors.New("barge-in received")
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
//
// This function is not affected by barge-out timeouts because results will
// still be returned in that case.
func (asrInteraction *AsrInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-asrInteraction.resultsReadyChannel:
		// resultsReadyChannel is closed when final results arrive
		return nil
	case <-asrInteraction.vadBargeInTimeoutChannel:
		// vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForNextResult waits for the next result-like response, whether that
// is a partial result, a final result, or a barge-in timeout. It returns true to
// indicate a final (interaction-ending) result and false to indicate a partial
// result. If a partial result is detected, resultIdx will indicate the index of
// that partial result.
//
// If the return indicates that a partial result has arrived, GetPartialResult
// can be used to fetch the result. Otherwise, GetFinalResults can be used to
// fetch the final result or error.
//
// If a result-like response does not arrive before the timeout, an error will
// be returned.
//
// This function is not affected by barge-out timeouts because results will
// still be returned in that case.
func (asrInteraction *AsrInteractionObject) WaitForNextResult(timeout time.Duration) (resultIdx int, final bool, err error) {

	// before doing anything else, get the index of the next partial result. this
	// will allow us to read from the correct channel, if we end up waiting.
	asrInteraction.partialResultLock.Lock()
	nextPartialResultIdx := asrInteraction.partialResultsReceived
	asrInteraction.partialResultLock.Unlock()

	// Before calling a select on multiple channels, try just the resultsReadyChannel.
	// If multiple channels in a select statement are available for reading at the
	// same moment, the chosen channel is randomly selected. We have the extra check
	// here so that we always indicate a final result if a final result has arrived.
	select {
	case <-asrInteraction.resultsReadyChannel:
		// resultsReadyChannel has already closed. Return (0, true) to indicate
		// that the final results have already arrived.
		return 0, true, nil
	default:
		// resultsReadyChannel has not been closed. Wait for the result below.
	}

	// The final result has not arrived. Wait for the next partial result or final result.
	select {
	case <-asrInteraction.resultsReadyChannel:
		// We received a final result. Return (0, true) to indicate that the
		// interaction is complete.
		return 0, true, nil
	case <-asrInteraction.partialResultsChannels[nextPartialResultIdx]:
		// We received a new partial result. Return the index, and false to
		// indicate a partial result.
		return nextPartialResultIdx, false, nil
	case <-asrInteraction.vadBargeInTimeoutChannel:
		// We received a barge-in timeout. Return (0, true) to indicate that the
		// interaction is complete.
		return 0, true, nil
	case <-time.After(timeout):
		return 0, false, TimeoutError
	}
}

// GetPartialResult returns a partial result at a given index. If a partial result does not exist
// at the given index, an error is returned.
//
// This function is best used in conjunction with WaitForNextResult, which will return the index of
// any new partial results.
func (asrInteraction *AsrInteractionObject) GetPartialResult(resultIdx int) (*api.PartialResult, error) {

	if resultIdx < 0 || len(asrInteraction.partialResultsList) <= resultIdx {
		return nil, errors.New("partial result index out of bounds")
	} else {
		return asrInteraction.partialResultsList[resultIdx], nil
	}
}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, like when a barge-in timeout is received, an error
// describing the issue will be returned.
//
// If the interaction does not end before the specified timeout, an error will
// be returned.
//
// This function is not affected by barge-out timeouts because results will
// still be returned in that case.
func (asrInteraction *AsrInteractionObject) GetFinalResults(timeout time.Duration) (result *api.AsrInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = asrInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if asrInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if asrInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_GRAMMAR_MATCH {
			// Successful interaction, return result
			return asrInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", asrInteraction.FinalResultStatus, asrInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else if asrInteraction.vadBargeInTimeoutReceived {
		// If we received a barge-in timeout, return an error.
		return nil, errors.New("barge-in timeout")
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of ASR interaction")
	}
}
