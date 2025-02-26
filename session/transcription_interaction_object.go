package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// TranscriptionInteractionObject represents a transcription interaction.
type TranscriptionInteractionObject struct {
	InteractionId string

	// VAD event tracking
	vadEventsLock              sync.Mutex
	vadBeginProcessingChannels []chan struct{}
	vadBargeInChannels         []chan struct{}
	vadBargeOutChannels        []chan struct{}
	vadBargeInTimeoutChannels  []chan struct{}
	vadRecordLog               []*vadInteractionRecord
	vadInteractionCounter      int
	vadCurrentState            api.VadEvent_VadEventType

	// Partial result tracking
	partialResultLock      sync.Mutex
	partialResultsReceived int
	partialResultsChannels []chan struct{}
	partialResultsList     []*api.PartialResult

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.TranscriptionInteractionResult
	resultsReadyChannel  chan struct{}

	// For special result handling
	isContinuousTranscription bool
}

// NewTranscription attempts to create a new transcription interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewTranscription(
	language string,
	embeddedGrammars []*api.Grammar,
	audioConsumeSettings *api.AudioConsumeSettings,
	normalizationSettings *api.NormalizationSettings,
	vadSettings *api.VadSettings,
	recognitionSettings *api.RecognitionSettings,
	languageModelName string,
	acousticModelName string,
	enablePostProcessing string,
	enableContinuousTranscription *api.OptionalBool) (interactionObject *TranscriptionInteractionObject, err error) {

	// Create transcription interaction, adding parameters such as VAD and recognition settings

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getTranscriptionRequest("", language, embeddedGrammars,
		vadSettings, audioConsumeSettings, normalizationSettings, recognitionSettings,
		languageModelName, acousticModelName, enablePostProcessing, enableContinuousTranscription))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateTranscriptionRequest error: %v", err)
		log.Printf("error sending transcription create request: %v", err.Error())
		return nil, err
	}

	// Get the interaction ID.
	transcriptionResponse := <-session.createdTranscriptionChannel
	interactionId := transcriptionResponse.InteractionId
	if EnableVerboseLogging {
		log.Printf("created new transcription interaction: %s", interactionId)
	}

	// determine if this is a continuous interaction
	isContinuousTranscription := false
	if enableContinuousTranscription != nil && enableContinuousTranscription.Value {
		isContinuousTranscription = true
	}

	// Create the interaction object.
	interactionObject = &TranscriptionInteractionObject{
		InteractionId:        interactionId,
		finalResultsReceived: false,
		finalResults:         nil,
		resultsReadyChannel:  make(chan struct{}),
		vadCurrentState:      api.VadEvent_VAD_EVENT_TYPE_UNSPECIFIED,

		isContinuousTranscription: isContinuousTranscription,
	}
	interactionObject.partialResultsChannels = append(interactionObject.partialResultsChannels, make(chan struct{}))
	interactionObject.vadBeginProcessingChannels = append(interactionObject.vadBeginProcessingChannels, make(chan struct{}))
	interactionObject.vadBargeInChannels = append(interactionObject.vadBargeInChannels, make(chan struct{}))
	interactionObject.vadBargeOutChannels = append(interactionObject.vadBargeOutChannels, make(chan struct{}))
	interactionObject.vadBargeInTimeoutChannels = append(interactionObject.vadBargeInTimeoutChannels, make(chan struct{}))
	interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())

	// Add the interaction object to the session
	session.transcriptionInteractionsMap[interactionId] = interactionObject

	return interactionObject, err
}

// WaitForBeginProcessing blocks until the API sends a VAD_BEGIN_PROCESSING
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// an error will be returned.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBeginProcessing(vadInteractionIdx int, timeout time.Duration) error {

	// Verify that the index is valid
	vadRecordLogLength := len(transcriptionInteraction.vadRecordLog)
	if vadInteractionIdx >= vadRecordLogLength || vadInteractionIdx < 0 {
		return errors.New("invalid VAD interaction index")
	}

	// With a valid index, check the VAD record log to see if we have already
	// received the BEGIN_PROCESSING event for this interaction.
	if transcriptionInteraction.vadRecordLog[vadInteractionIdx].beginProcessingReceived {
		return nil
	}

	// Otherwise, wait for the BEGIN_PROCESSING event to arrive.
	select {
	case <-transcriptionInteraction.vadBeginProcessingChannels[vadInteractionIdx]:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeIn blocks until the API sends a VAD_BARGE_IN message. If the
// message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, an error will be returned.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBargeIn(vadInteractionIdx int, timeout time.Duration) error {

	// TODO: instead of just checking the barge_in channel, we should also pay attention
	//       to other events that may preclude a barge_in. For example, if a barge_in_timeout
	//       arrives for the relevant VAD interaction, we can be reasonably confident that a
	//       barge_in will never arrive.

	// Verify that the index is valid
	vadRecordLogLength := len(transcriptionInteraction.vadRecordLog)
	if vadInteractionIdx >= vadRecordLogLength || vadInteractionIdx < 0 {
		return errors.New("invalid VAD interaction index")
	}

	// With a valid index, check the VAD record log to see if we have already
	// received a BARGE_IN event for this interaction.
	// NOTE: valid barge-ins should not have a value of 0, so we use this to
	// check if a barge-in has arrived or not.
	if transcriptionInteraction.vadRecordLog[vadInteractionIdx].bargeInReceived > 0 {
		return nil
	}

	// Otherwise, wait for the BARGE_IN event to arrive.
	select {
	case <-transcriptionInteraction.vadBargeInChannels[vadInteractionIdx]:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForEndOfSpeech blocks until the API sends a VAD_END_OF_SPEECH message. If
// the message has already arrived, this function returns immediately. Otherwise,
// if the message does not return before the timeout, an error will be returned.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForEndOfSpeech(vadInteractionIdx int, timeout time.Duration) error {

	// Verify that the index is valid
	vadRecordLogLength := len(transcriptionInteraction.vadRecordLog)
	if vadInteractionIdx >= vadRecordLogLength || vadInteractionIdx < 0 {
		return errors.New("invalid VAD interaction index")
	}

	// With a valid index, check the VAD record log to see if we have already
	// received a BARGE_OUT event for this interaction.
	// NOTE: valid barge-outs should not have a value of 0, so we use this to
	// check if a barge-out has arrived or not.
	if transcriptionInteraction.vadRecordLog[vadInteractionIdx].bargeOutReceived > 0 {
		return nil
	}

	// Otherwise, wait for the BARGE_OUT event to arrive.
	select {
	case <-transcriptionInteraction.vadBargeOutChannels[vadInteractionIdx]:
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForBargeInTimeout blocks until the API sends a VAD_BARGE_IN_TIMEOUT
// message. If the message has already arrived, this function returns
// immediately. Otherwise, if the message does not return before the timeout,
// an error will be returned.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBargeInTimeout(vadInteractionIdx int, timeout time.Duration) error {

	// Verify that the index is valid
	vadRecordLogLength := len(transcriptionInteraction.vadRecordLog)
	if vadInteractionIdx >= vadRecordLogLength || vadInteractionIdx < 0 {
		return errors.New("invalid VAD interaction index")
	}

	// With a valid index, check the VAD record log to see if we have already
	// received a BARGE_IN_TIMEOUT event for this interaction.
	if transcriptionInteraction.vadRecordLog[vadInteractionIdx].bargeInTimeoutReceived {
		return nil
	}

	// Otherwise, wait for the BARGE_OUT event to arrive.
	select {
	case <-transcriptionInteraction.vadBargeInTimeoutChannels[vadInteractionIdx]:
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
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	if transcriptionInteraction.isContinuousTranscription {
		select {
		case <-transcriptionInteraction.resultsReadyChannel:
			// resultsReadyChannel is closed when final results arrive
			return nil
		case <-time.After(timeout):
			return TimeoutError
		}
	} else {
		select {
		case <-transcriptionInteraction.resultsReadyChannel:
			// resultsReadyChannel is closed when final results arrive
			return nil
		case <-transcriptionInteraction.vadBargeInTimeoutChannels[0]:
			// vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
			return nil
		case <-time.After(timeout):
			return TimeoutError
		}
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
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForNextResult(timeout time.Duration) (resultIdx int, final bool, err error) {

	// before doing anything else, get the index of the next partial result. this
	// will allow us to read from the correct channel, if we end up waiting.
	transcriptionInteraction.partialResultLock.Lock()
	nextPartialResultIdx := transcriptionInteraction.partialResultsReceived
	transcriptionInteraction.partialResultLock.Unlock()

	// Before calling a select on multiple channels, try just the resultsReadyChannel.
	// If multiple channels in a select statement are available for reading at the
	// same moment, the chosen channel is randomly selected. We have the extra check
	// here so that we always indicate a final result if a final result has arrived.
	select {
	case <-transcriptionInteraction.resultsReadyChannel:
		// resultsReadyChannel has already closed. Return (0, true) to indicate
		// that the final results have already arrived
		return 0, true, nil
	default:
		// resultsReadyChannel has not been closed. Wait for the result below.
	}

	// The final result has not arrived. Wait for the next partial result or final result.
	if transcriptionInteraction.isContinuousTranscription {

		select {
		case <-transcriptionInteraction.resultsReadyChannel:
			// We received a final result. Return (0, true) to indicate that the
			// interaction is complete.
			return 0, true, nil
		case <-transcriptionInteraction.partialResultsChannels[nextPartialResultIdx]:
			// We received a new partial result. Return the index, and false to
			// indicate a partial result.
			return nextPartialResultIdx, false, nil
		case <-time.After(timeout):
			// We didn't get any result-like responses. Return an error.
			return 0, false, TimeoutError
		}
	} else {
		select {
		case <-transcriptionInteraction.resultsReadyChannel:
			// We received a final result. Return (0, true) to indicate that the
			// interaction is complete.
			return 0, true, nil
		case <-transcriptionInteraction.partialResultsChannels[nextPartialResultIdx]:
			// We received a new partial result. Return the index, and false to
			// indicate a partial result.
			return nextPartialResultIdx, false, nil
		case <-transcriptionInteraction.vadBargeInTimeoutChannels[0]:
			// We received a barge-in timeout. Return (0, true) to indicate that the
			// interaction is complete.
			return 0, true, nil
		case <-time.After(timeout):
			// We didn't get any result-like responses. Return an error.
			return 0, false, TimeoutError
		}
	}
}

// GetPartialResult returns a partial result at a given index. If a partial result does not exist
// at the given index, an error is returned.
//
// This function is best used in conjunction with WaitForNextResult, which will return the index of
// any new partial results.
func (transcriptionInteraction *TranscriptionInteractionObject) GetPartialResult(resultIdx int) (*api.PartialResult, error) {

	if resultIdx < 0 || len(transcriptionInteraction.partialResultsList) <= resultIdx {
		return nil, errors.New("partial result index out of bounds")
	} else {
		return transcriptionInteraction.partialResultsList[resultIdx], nil
	}
}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, like when a barge-in timeout is received, an error
// describing the issue will be returned.
//
// If the interaction does not end before the specified timeout, an error will
// be returned.
func (transcriptionInteraction *TranscriptionInteractionObject) GetFinalResults(timeout time.Duration) (*api.TranscriptionInteractionResult, error) {

	// Wait for the end of the interaction.
	err := transcriptionInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if transcriptionInteraction.finalResultsReceived {
		// If we received final results, return them.
		return transcriptionInteraction.finalResults, nil
	} else if transcriptionInteraction.isContinuousTranscription == false && transcriptionInteraction.vadRecordLog[0].bargeInTimeoutReceived {
		// If we received a barge-in timeout, return an error.
		return nil, errors.New("barge-in timeout")
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of transcription interaction")
	}
}
