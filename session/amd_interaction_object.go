package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
	"fmt"
	"log"
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
	resultsReadyChannel  chan struct{}
}

// NewAmd attempts to create a new AMD interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewAmd(
	amdSettings *api.AmdSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	vadSettings *api.VadSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *AmdInteractionObject, err error) {

	// Create AMD interaction, adding parameters such as VAD and recognition settings

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getAmdRequest("",
		amdSettings, audioConsumeSettings, vadSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateAmdRequest error: %v", err)
		log.Printf("error sending AMD create request: %v", err.Error())
		return nil, err
	}

	// Get the interaction ID.
	amdResponse := <-session.createdAmdChannel
	interactionId := amdResponse.InteractionId
	if EnableVerboseLogging {
		log.Printf("created new AMD interaction: %s", interactionId)
	}

	// Create the interaction object.
	interactionObject = &AmdInteractionObject{
		InteractionId:             interactionId,
		vadBeginProcessingChannel: make(chan struct{}, 1),
		vadBargeInChannel:         make(chan int, 1),
		vadBargeInReceived:        -1,
		vadBargeOutChannel:        make(chan int, 1),
		vadBargeOutReceived:       -1,
		vadBargeInTimeoutChannel:  make(chan struct{}),
		vadBargeInTimeoutReceived: false,
		finalResultsReceived:      false,
		finalResults:              nil,
		resultsReadyChannel:       make(chan struct{}),
	}

	// Add the interaction object to the session
	session.amdInteractionsMap[interactionId] = interactionObject

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

//// WaitForNextResult waits for the next result-like response, whether that
//// is a partial result, a final result, or a barge-in timeout. It returns true to
//// indicate a final (interaction-ending) result and false to indicate a partial
//// result. If a partial result is detected, resultIdx will indicate the index of
//// that partial result.
////
//// If the return indicates that a partial result has arrived, GetPartialResult
//// can be used to fetch the result. Otherwise, GetFinalResults can be used to
//// fetch the final result or error.
////
//// If a result-like response does not arrive before the timeout, an error will
//// be returned.
//func (amdInteraction *AmdInteractionObject) WaitForNextResult(timeout time.Duration) (resultIdx int, final bool, err error) {
//
//	// before doing anything else, get the index of the next partial result. this
//	// will allow us to read from the correct channel, if we end up waiting.
//	amdInteraction.partialResultLock.Lock()
//	nextPartialResultIdx := amdInteraction.partialResultsReceived
//	amdInteraction.partialResultLock.Unlock()
//
//	// Before calling a select on multiple channels, try just the resultsReadyChannel.
//	// If multiple channels in a select statement are available for reading at the
//	// same moment, the chosen channel is randomly selected. We have the extra check
//	// here so that we always indicate a final result if a final result has arrived.
//	select {
//	case <-amdInteraction.resultsReadyChannel:
//		// resultsReadyChannel has already closed. Return (0, true) to indicate
//		// that the final results have already arrived.
//		return 0, true, nil
//	default:
//		// resultsReadyChannel has not been closed. Wait for the result below.
//	}
//
//	// The final result has not arrived. Wait for the next partial result or final result.
//	select {
//	case <-amdInteraction.resultsReadyChannel:
//		// We received a final result. Return (0, true) to indicate that the
//		// interaction is complete.
//		return 0, true, nil
//	case <-amdInteraction.partialResultsChannels[nextPartialResultIdx]:
//		// We received a new partial result. Return the index, and false to
//		// indicate a partial result.
//		return nextPartialResultIdx, false, nil
//	case <-amdInteraction.vadBargeInTimeoutChannel:
//		// We received a barge-in timeout. Return (0, true) to indicate that the
//		// interaction is complete.
//		return 0, true, nil
//	case <-time.After(timeout):
//		return 0, false, TimeoutError
//	}
//}

//// GetPartialResult returns a partial result at a given index. If a partial result does not exist
//// at the given index, an error is returned.
////
//// This function is best used in conjunction with WaitForNextResult, which will return the index of
//// any new partial results.
//func (amdInteraction *AmdInteractionObject) GetPartialResult(resultIdx int) (*api.PartialResult, error) {
//
//	if resultIdx < 0 || len(amdInteraction.partialResultsList) <= resultIdx {
//		return nil, errors.New("partial result index out of bounds")
//	} else {
//		return amdInteraction.partialResultsList[resultIdx], nil
//	}
//}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, like when a barge-in timeout is received, an error
// describing the issue will be returned.
//
// If the interaction does not end before the specified timeout, an error will
// be returned.
func (amdInteraction *AmdInteractionObject) GetFinalResults(timeout time.Duration) (*api.AmdInteractionResult, error) {

	// Wait for the end of the interaction.
	err := amdInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if amdInteraction.finalResultsReceived {
		// If we received final results, return them.
		return amdInteraction.finalResults, nil
	} else if amdInteraction.vadBargeInTimeoutReceived {
		// If we received a barge-in timeout, return an error.
		return nil, errors.New("barge-in timeout")
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of AMD interaction")
	}
}
