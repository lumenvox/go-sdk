package session

import (
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "errors"
)

// WaitForBeginProcessing blocks until the API sends a VAD_BEGIN_PROCESSING
// message. If the message has already arrived, this function returns
// immediately.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBeginProcessing() {

    if transcriptionInteraction.vadBeginProcessingReceived {
        return
    }

    <-transcriptionInteraction.vadBeginProcessingChannel
}

// WaitForBargeIn blocks until the API sends a VAD_BARGE_IN message. If the
// message has already arrived, this function returns immediately.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBargeIn() {

    if transcriptionInteraction.vadBargeInReceived != -1 {
        return
    }

    <-transcriptionInteraction.vadBargeInChannel
}

// WaitForEndOfSpeech blocks until the API sends a VAD_END_OF_SPEECH message. If
// the message has already arrived, this function returns immediately.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForEndOfSpeech() {

    if transcriptionInteraction.vadBargeOutReceived != -1 {
        return
    }

    <-transcriptionInteraction.vadBargeOutChannel
}

// WaitForBargeInTimeout blocks until the API sends a VAD_BARGE_IN_TIMEOUT
// message. If the message has already arrived, this function returns
// immediately.
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBargeInTimeout() {

    if transcriptionInteraction.vadBargeInTimeoutReceived {
        return
    }

    <-transcriptionInteraction.vadBargeInTimeoutChannel
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors (like a
// barge-in timeout).
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForFinalResults() {

    select {
    case <-transcriptionInteraction.resultsReadyChannel:
        // resultsReadyChannel is closed when final results arrive
    case <-transcriptionInteraction.vadBargeInTimeoutChannel:
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
func (transcriptionInteraction *TranscriptionInteractionObject) WaitForNextResult() (resultIdx int, final bool) {

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
        return 0, true
    default:
        // resultsReadyChannel has not been closed. Wait for the result below.
    }

    // The final result has not arrived. Wait for the next partial result or final result.
    select {
    case <-transcriptionInteraction.resultsReadyChannel:
        // We received a final result. Return (0, true) to indicate that the
        // interaction is complete.
        return 0, true
    case <-transcriptionInteraction.partialResultsChannels[nextPartialResultIdx]:
        // We received a new partial result. Return the index, and false to
        // indicate a partial result.
        return nextPartialResultIdx, false
    case <-transcriptionInteraction.vadBargeInTimeoutChannel:
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
func (transcriptionInteraction *TranscriptionInteractionObject) GetFinalResults() (*api.TranscriptionInteractionResult, error) {

    // Wait for the end of the interaction.
    transcriptionInteraction.WaitForFinalResults()

    if transcriptionInteraction.finalResultsReceived {
        // If we received final results, return them.
        return transcriptionInteraction.finalResults, nil
    } else if transcriptionInteraction.vadBargeInTimeoutReceived {
        // If we received a barge-in timeout, return an error.
        return nil, errors.New("barge-in timeout")
    } else {
        // This should never happen.
        return nil, errors.New("unexpected end of transcription interaction")
    }
}
