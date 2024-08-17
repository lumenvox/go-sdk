package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"errors"
)

// WaitForFinalResults waits for the final results.
func (ttsInteraction *TtsInteractionObject) WaitForFinalResults() {

	// ResultsReadyChannel is closed when final results arrive.
	<-ttsInteraction.resultsReadyChannel
}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, an error describing the issue will be returned.
func (ttsInteraction *TtsInteractionObject) GetFinalResults() (*api.TtsInteractionResult, error) {

	// Wait for the end of the interaction.
	ttsInteraction.WaitForFinalResults()

	if ttsInteraction.finalResultsReceived {
		// If we received final results, return them.
		return ttsInteraction.finalResults, nil
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of tts interaction")
	}
}
