package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"errors"
)

func (normalizationInteraction *NormalizationInteractionObject) WaitForFinalResults() {

	<-normalizationInteraction.resultsReadyChannel
}

func (normalizationInteraction *NormalizationInteractionObject) GetFinalResults() (*api.NormalizeTextResult, error) {

	normalizationInteraction.WaitForFinalResults()

	if normalizationInteraction.finalResultsReceived {
		return normalizationInteraction.finalResults, nil
	} else {
		return nil, errors.New("unexpected end of normalization interaction")
	}
}
