package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
)

// SentimentInteractionObject represents a sentiment interaction.
type SentimentInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.SentimentInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateSentimentHelper struct {
	interactionCreateChannel chan *SentimentInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateSentiment creates a helper in an internal map to prepare
// for an interactionCreateSentiment response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateSentiment(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *SentimentInteractionObject, err error) {

	session.interactionCreateSentimentMapLock.Lock()
	defer session.interactionCreateSentimentMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateSentimentMap[correlationId]; ok {
		return nil, errors.New("interaction create sentiment channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateSentimentHelper{
		interactionCreateChannel: make(chan *SentimentInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateSentimentMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewSentiment attempts to create a new sentiment interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewSentiment(language string,
	textToAnalyze []string,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *SentimentInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateSentiment(correlationId, interactionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getSentimentRequest(correlationId, language, textToAnalyze,
		generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateSentimentRequest error: %v", err)
		logger.Error("sending sentiment create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(interactionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "itn",
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "itn",
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new sentiment interaction",
			"interaction_id", interactionObject.InteractionId)
	}

	return interactionObject, err
}

// WaitForFinalResults waits for the end of the interaction. This is typically
// triggered by final results, but it can also be triggered by errors.
//
// If nothing arrives before the timeout, an error will be returned. Note that
// interaction failures do not trigger errors from this function, so long as
// the notification arrives before the timeout.
func (sentimentInteraction *SentimentInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-sentimentInteraction.resultsReadyChannel:
		// resultsReadyChannel is closed when final results arrive
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// GetFinalResults fetches the final results or error from an interaction,
// waiting if necessary. If the interaction succeeds, results will be returned.
// In other cases, like when a barge-in timeout is received, an error
// describing the issue will be returned.
//
// If the interaction does not end before the specified timeout, an error will
// be returned.
func (sentimentInteraction *SentimentInteractionObject) GetFinalResults(timeout time.Duration) (result *api.SentimentInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = sentimentInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if sentimentInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if sentimentInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_SENTIMENT_RESULT {
			// Successful interaction, return result
			return sentimentInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", sentimentInteraction.FinalResultStatus, sentimentInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of sentiment interaction")
	}
}
