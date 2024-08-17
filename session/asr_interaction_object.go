package session

import (
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "errors"
)

// WaitForBeginProcessing blocks until the API sends a VAD_BEGIN_PROCESSING
// message. If the message has already arrived, this function returns
// immediately.
func (asrInteraction *AsrInteractionObject) WaitForBeginProcessing() {

    if asrInteraction.vadBeginProcessingReceived {
        return
    }

    <-asrInteraction.vadBeginProcessingChannel
}

// WaitForBargeIn blocks until the API sends a VAD_BARGE_IN message. If the
// message has already arrived, this function returns immediately.
func (asrInteraction *AsrInteractionObject) WaitForBargeIn() {

    if asrInteraction.vadBargeInReceived != -1 {
        return
    }

    <-asrInteraction.vadBargeInChannel
}

// WaitForEndOfSpeech blocks until the API sends a VAD_END_OF_SPEECH message. If
// the message has already arrived, this function returns immediately.
func (asrInteraction *AsrInteractionObject) WaitForEndOfSpeech() {

    if asrInteraction.vadBargeOutReceived != -1 {
        return
    }

    <-asrInteraction.vadBargeOutChannel
}

// WaitForBargeInTimeout blocks until the API sends a VAD_BARGE_IN_TIMEOUT
// message. If the message has already arrived, this function returns
// immediately.
func (asrInteraction *AsrInteractionObject) WaitForBargeInTimeout() {

    if asrInteraction.vadBargeInTimeoutReceived {
        return
    }

    // vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
    <-asrInteraction.vadBargeInTimeoutChannel
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors (like a
// barge-in timeout).
func (asrInteraction *AsrInteractionObject) WaitForFinalResults() {

    select {
    case <-asrInteraction.resultsReadyChannel:
        // resultsReadyChannel is closed when final results arrive
    case <-asrInteraction.vadBargeInTimeoutChannel:
        // vadBargeInTimeoutChannel is closed when a barge-in timeout arrives
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
func (asrInteraction *AsrInteractionObject) WaitForNextResult() (resultIdx int, final bool) {

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
        return 0, true
    default:
        // resultsReadyChannel has not been closed. Wait for the result below.
    }

    // The final result has not arrived. Wait for the next partial result or final result.
    select {
    case <-asrInteraction.resultsReadyChannel:
        // We received a final result. Return (0, true) to indicate that the
        // interaction is complete.
        return 0, true
    case <-asrInteraction.partialResultsChannels[nextPartialResultIdx]:
        // We received a new partial result. Return the index, and false to
        // indicate a partial result.
        return nextPartialResultIdx, false
    case <-asrInteraction.vadBargeInTimeoutChannel:
        // We received a barge-in timeout. Return (0, true) to indicate that the
        // interaction is complete.
        return 0, true
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
func (asrInteraction *AsrInteractionObject) GetFinalResults() (*api.AsrInteractionResult, error) {

    // Wait for the end of the interaction.
    asrInteraction.WaitForFinalResults()

    if asrInteraction.finalResultsReceived {
        // If we received final results, return them.
        return asrInteraction.finalResults, nil
    } else if asrInteraction.vadBargeInTimeoutReceived {
        // If we received a barge-in timeout, return an error.
        return nil, errors.New("barge-in timeout")
    } else {
        // This should never happen.
        return nil, errors.New("unexpected end of ASR interaction")
    }
}
