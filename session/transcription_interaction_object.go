package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"errors"
)

func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBeginProcessing() {

	if transcriptionInteraction.vadBeginProcessingReceived {
		return
	}

	<-transcriptionInteraction.vadBeginProcessingChannel
}

func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBargeIn() {

	if transcriptionInteraction.vadBargeInReceived != -1 {
		return
	}

	<-transcriptionInteraction.vadBargeInChannel
}

func (transcriptionInteraction *TranscriptionInteractionObject) WaitForEndOfSpeech() {

	if transcriptionInteraction.vadBargeOutReceived != -1 {
		return
	}

	<-transcriptionInteraction.vadBargeOutChannel
}

func (transcriptionInteraction *TranscriptionInteractionObject) WaitForBargeInTimeout() {

	if transcriptionInteraction.vadBargeInTimeoutReceived {
		return
	}

	<-transcriptionInteraction.vadBargeInTimeoutChannel
}

func (transcriptionInteraction *TranscriptionInteractionObject) WaitForFinalResults() {

	select {
	case <-transcriptionInteraction.resultsReadyChannel:
	case <-transcriptionInteraction.vadBargeInTimeoutChannel:
	}
}

func (transcriptionInteraction *TranscriptionInteractionObject) WaitForNextResult() (resultIdx int, final bool) {

	transcriptionInteraction.partialResultLock.Lock()
	nextPartialResultIdx := transcriptionInteraction.partialResultsReceived
	transcriptionInteraction.partialResultLock.Unlock()

	select {
	case <-transcriptionInteraction.resultsReadyChannel:
		return 0, true
	default:
	}

	select {
	case <-transcriptionInteraction.resultsReadyChannel:
		return 0, true
	case <-transcriptionInteraction.partialResultsChannels[nextPartialResultIdx]:
		return nextPartialResultIdx, false
	case <-transcriptionInteraction.vadBargeInTimeoutChannel:
		return 0, true
	}
}

func (transcriptionInteraction *TranscriptionInteractionObject) GetPartialResult(resultIdx int) (*api.PartialResult, error) {

	if resultIdx < 0 || len(transcriptionInteraction.partialResultsList) <= resultIdx {
		return nil, errors.New("partial result index out of bounds")
	} else {
		return transcriptionInteraction.partialResultsList[resultIdx], nil
	}
}

func (transcriptionInteraction *TranscriptionInteractionObject) GetFinalResults() (*api.TranscriptionInteractionResult, error) {

	transcriptionInteraction.WaitForFinalResults()

	if transcriptionInteraction.finalResultsReceived {
		return transcriptionInteraction.finalResults, nil
	} else if transcriptionInteraction.vadBargeInTimeoutReceived {
		return nil, errors.New("barge-in timeout")
	} else {
		return nil, errors.New("unexpected end of transcription interaction")
	}
}
