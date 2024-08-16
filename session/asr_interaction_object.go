package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"errors"
)

func (asrInteraction *AsrInteractionObject) WaitForBeginProcessing() {

	if asrInteraction.vadBeginProcessingReceived {
		return
	}

	<-asrInteraction.vadBeginProcessingChannel
}

func (asrInteraction *AsrInteractionObject) WaitForBargeIn() {

	if asrInteraction.vadBargeInReceived != -1 {
		return
	}

	<-asrInteraction.vadBargeInChannel
}

func (asrInteraction *AsrInteractionObject) WaitForEndOfSpeech() {

	if asrInteraction.vadBargeOutReceived != -1 {
		return
	}

	<-asrInteraction.vadBargeOutChannel
}

func (asrInteraction *AsrInteractionObject) WaitForBargeInTimeout() {

	if asrInteraction.vadBargeInTimeoutReceived {
		return
	}

	<-asrInteraction.vadBargeInTimeoutChannel
}

func (asrInteraction *AsrInteractionObject) WaitForFinalResults() {

	select {
	case <-asrInteraction.resultsReadyChannel:
	case <-asrInteraction.vadBargeInTimeoutChannel:
	}
}

func (asrInteraction *AsrInteractionObject) WaitForNextResult() (resultIdx int, final bool) {

	asrInteraction.partialResultLock.Lock()
	nextPartialResultIdx := asrInteraction.partialResultsReceived
	asrInteraction.partialResultLock.Unlock()

	select {
	case <-asrInteraction.resultsReadyChannel:
		return 0, true
	default:
	}

	select {
	case <-asrInteraction.resultsReadyChannel:
		return 0, true
	case <-asrInteraction.partialResultsChannels[nextPartialResultIdx]:
		return nextPartialResultIdx, false
	case <-asrInteraction.vadBargeInTimeoutChannel:
		return 0, true
	}
}

func (asrInteraction *AsrInteractionObject) GetPartialResult(resultIdx int) (*api.PartialResult, error) {

	if resultIdx < 0 || len(asrInteraction.partialResultsList) <= resultIdx {
		return nil, errors.New("partial result index out of bounds")
	} else {
		return asrInteraction.partialResultsList[resultIdx], nil
	}
}

func (asrInteraction *AsrInteractionObject) GetFinalResults() (*api.AsrInteractionResult, error) {

	asrInteraction.WaitForFinalResults()

	if asrInteraction.finalResultsReceived {
		return asrInteraction.finalResults, nil
	} else if asrInteraction.vadBargeInTimeoutReceived {
		return nil, errors.New("barge-in timeout")
	} else {
		return nil, errors.New("unexpected end of ASR interaction")
	}
}
