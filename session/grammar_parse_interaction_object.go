package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
)

// GrammarParseInteractionObject represents a grammar parse interaction.
type GrammarParseInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.GrammarParseInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateGrammarParseHelper struct {
	interactionCreateChannel chan *GrammarParseInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateGrammarParse creates a helper in an internal map to prepare
// for an interactionCreateGrammarParse response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateGrammarParse(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *GrammarParseInteractionObject, err error) {

	session.interactionCreateGrammarParseMapLock.Lock()
	defer session.interactionCreateGrammarParseMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateGrammarParseMap[correlationId]; ok {
		return nil, errors.New("interaction create grammar parse channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateGrammarParseHelper{
		interactionCreateChannel: make(chan *GrammarParseInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateGrammarParseMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewGrammarParse attempts to create a new grammar parse interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewGrammarParse(language string,
	inputText string,
	grammars []*api.Grammar,
	grammarSettings *api.GrammarSettings,
	parseTimeoutMs *api.OptionalInt32,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *GrammarParseInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateGrammarParse(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getGrammarParseRequest(correlationId, language, inputText, grammars,
		grammarSettings, parseTimeoutMs, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateGrammarParseRequest error: %v", err)
		logger.Error("sending grammar parse create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "grammar parse",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "grammar parse",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new grammar parse interaction",
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
func (grammarParseInteraction *GrammarParseInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-grammarParseInteraction.resultsReadyChannel:
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
func (grammarParseInteraction *GrammarParseInteractionObject) GetFinalResults(timeout time.Duration) (result *api.GrammarParseInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = grammarParseInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if grammarParseInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if grammarParseInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_GRAMMAR_MATCH {
			// Successful interaction, return result
			return grammarParseInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", grammarParseInteraction.FinalResultStatus, grammarParseInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of grammar parse interaction")
	}
}
