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

// TtsInteractionObject represents a TTS interaction.
type TtsInteractionObject struct {
	InteractionId string

	// Partial result tracking
	partialResultLock      sync.Mutex
	partialResultsReceived int
	partialResultsChannels []chan struct{}
	partialResultsList     []*api.PartialResult

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.TtsInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateTtsHelper struct {
	interactionCreateChannel chan *TtsInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateTts creates a helper in an internal map to prepare
// for an interactionCreateTts response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateTts(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *TtsInteractionObject, err error) {

	session.interactionCreateTtsMapLock.Lock()
	defer session.interactionCreateTtsMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateTtsMap[correlationId]; ok {
		return nil, errors.New("interaction create tts channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateTtsHelper{
		interactionCreateChannel: make(chan *TtsInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateTtsMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewInlineTts attempts to create a new inline TTS interaction. If successful,
// a new interaction object will be returned.
func (session *SessionObject) NewInlineTts(language string,
	textToSynthesize string,
	inlineSettings *api.TtsInlineSynthesisSettings,
	synthesizedAudioFormat *api.AudioFormat,
	synthesisTimeoutMs *api.OptionalInt32,
	generalInteractionSettings *api.GeneralInteractionSettings,
	enablePartialResults *api.OptionalBool) (interactionObject *TtsInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateTts(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getInlineTtsRequest(correlationId, language, textToSynthesize,
		synthesizedAudioFormat, synthesisTimeoutMs, inlineSettings, generalInteractionSettings,
		enablePartialResults))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
		logger.Error("sending tts create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "tts",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "tts",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new TTS interaction",
			"interactionId", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// NewUrlTts attempts to create a new URL-based TTS interaction. If successful, a new interaction
// object will be returned.
func (session *SessionObject) NewUrlTts(language string,
	ssmlUrl string,
	sslVerifyPeer *api.OptionalBool,
	synthesizedAudioFormat *api.AudioFormat,
	synthesisTimeoutMs *api.OptionalInt32,
	generalInteractionSettings *api.GeneralInteractionSettings,
	enablePartialResults *api.OptionalBool) (interactionObject *TtsInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateTts(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getUrlTtsRequest(correlationId, language, ssmlUrl,
		synthesizedAudioFormat, synthesisTimeoutMs, sslVerifyPeer, generalInteractionSettings,
		enablePartialResults))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
		logger.Error("sending tts create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "tts",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "tts",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new TTS interaction",
			"interactionId", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors.
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures do not trigger errors from this function, so long as
// the notification arrives before the timeout.
func (ttsInteraction *TtsInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-ttsInteraction.resultsReadyChannel:
		// resultsReadyChannel is closed when final results arrive
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// WaitForNextResult waits for the next result-like response, whether that is
// a partial result or a final result. It returnes true to indicate a final
// result and falso to indicate a partial result. If a partial result is detected,
// resultIdx will indicate the index of that partial result.
//
// If the return indicates that a partial result was returned, GetPartialResult
// can be used to fetch the result. Otherwise, GetFinalResults can be used to
// fetch the final result.
//
// If a result-like response does not arrive before the timeout, an error will
// be returned.
func (ttsInteraction *TtsInteractionObject) WaitForNextResult(timeout time.Duration) (resultIdx int, final bool, err error) {

	// before doing anything else, get the index of the next partial result. this
	// will allow us to read from the correct channel if we end up waiting.
	ttsInteraction.partialResultLock.Lock()
	nextPartialResultIdx := ttsInteraction.partialResultsReceived
	ttsInteraction.partialResultLock.Unlock()

	// Before calling a select on multiple channels, try just the resultsReadyChannel.
	// If multiple channels in a select statement are available for reading at the
	// same moment, the chosen channel is randomly selected. We have the extra check
	// here so that we always indicate a final result if a final result has arrived.
	select {
	case <-ttsInteraction.resultsReadyChannel:
		return 0, true, nil
	default:
		// resultsReadyChannel has not been closed. Wait for the next result below.
	}

	// The final result has not arrived. Wait for the next partial result or final result.
	select {
	case <-ttsInteraction.resultsReadyChannel:
		// We received a final result. Return (0, true) to indicate that the
		// interaction is complete.
		return 0, true, nil
	case <-ttsInteraction.partialResultsChannels[nextPartialResultIdx]:
		// We received a new partial result. Return the index, and false to
		// indicate a partial result.
		return nextPartialResultIdx, false, nil
	case <-time.After(timeout):
		// We didn't get any result-like responses. Return an error.
		return 0, false, TimeoutError
	}
}

// GetPartialResult returns a partial result at a given index. If a partial result does not exist
// at the given index, an error is returned.
//
// This function is best used in conjunction with WaitForNextResult, which will return the index of
// any new partial results.
func (ttsInteraction *TtsInteractionObject) GetPartialResult(resultIdx int) (*api.PartialResult, error) {

	if resultIdx < 0 || len(ttsInteraction.partialResultsList) <= resultIdx {
		return nil, errors.New("partial result index out of bounds")
	} else {
		return ttsInteraction.partialResultsList[resultIdx], nil
	}
}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, an error describing the issue will be returned.
//
// If the interaction does not end before the specified timeout, an error will
// be returned.
func (ttsInteraction *TtsInteractionObject) GetFinalResults(timeout time.Duration) (result *api.TtsInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = ttsInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if ttsInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if ttsInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_TTS_READY {
			// Successful interaction, return result
			return ttsInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", ttsInteraction.FinalResultStatus, ttsInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of tts interaction")
	}
}
