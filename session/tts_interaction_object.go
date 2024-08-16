package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"errors"
)

func (ttsInteraction *TtsInteractionObject) WaitForFinalResults() {

	<-ttsInteraction.resultsReadyChannel
}

func (ttsInteraction *TtsInteractionObject) GetFinalResults() (*api.TtsInteractionResult, error) {

	ttsInteraction.WaitForFinalResults()

	if ttsInteraction.finalResultsReceived {
		return ttsInteraction.finalResults, nil
	} else {
		return nil, errors.New("unexpected end of tts interaction")
	}
}
