package session

import (
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "errors"
)

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors.
func (normalizationInteraction *NormalizationInteractionObject) WaitForFinalResults() {

    // resultsReadyChannel is closed when final results arrive
    <-normalizationInteraction.resultsReadyChannel
}

// GetFinalResults can be used to wait for the end of the interaction. If the
// interaction succeeds, results will be returned. Otherwise, a relevant error
// will be returned.
func (normalizationInteraction *NormalizationInteractionObject) GetFinalResults() (*api.NormalizeTextResult, error) {

    // Wait for the end of the interaction.
    normalizationInteraction.WaitForFinalResults()

    if normalizationInteraction.finalResultsReceived {
        // If we received final results, return them.
        return normalizationInteraction.finalResults, nil
    } else {
        // This should never happen.
        return nil, errors.New("unexpected end of normalization interaction")
    }
}
