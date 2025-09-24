package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
)

// NeuronInteractionObject represents a neuron interaction.
type NeuronInteractionObject struct {
	InteractionId string

	// Final result tracking
	finalResultsReceived bool
	finalResults         *api.NeuronInteractionResult
	FinalStatus          *status.Status
	FinalResultStatus    api.FinalResultStatus
	resultsReadyChannel  chan struct{}
}

type interactionCreateNeuronHelper struct {
	interactionCreateChannel chan *NeuronInteractionObject
	deadline                 time.Time
}

// prepareInteractionCreateNeuron creates a helper in an internal map to prepare
// for an interactionCreateNeuron response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionCreateNeuron(correlationId string, deadline time.Duration) (
	interactionCreateChan <-chan *NeuronInteractionObject, err error) {

	session.interactionCreateNeuronMapLock.Lock()
	defer session.interactionCreateNeuronMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionCreateNeuronMap[correlationId]; ok {
		return nil, errors.New("interaction create neuron channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	interactionCreateHelper := &interactionCreateNeuronHelper{
		interactionCreateChannel: make(chan *NeuronInteractionObject, 1),
		deadline:                 time.Now().Add(deadline),
	}
	session.interactionCreateNeuronMap[correlationId] = interactionCreateHelper

	interactionCreateChan = interactionCreateHelper.interactionCreateChannel

	return
}

// NewNeuron attempts to create a new neuron interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewNeuron(language string,
	textToProcess string,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *NeuronInteractionObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create a helper struct to wait for the new interaction object
	interactionCreateChan, err := session.prepareInteractionCreateNeuron(correlationId, InteractionCreateDeadline)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the interaction create request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getNeuronRequest(correlationId, language, textToProcess,
		generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateNeuronRequest error: %v", err)
		logger.Error("sending neuron create request",
			"error", err.Error())
		return nil, err
	}

	// Wait for the new interaction object.
	select {
	case interactionObject = <-interactionCreateChan:
	case <-time.After(InteractionCreateDeadline):
		logger.Error("timed out waiting for interaction object",
			"type", "itn",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("timed out waiting for interaction object")
	}

	if interactionObject == nil {
		logger.Error("received nil interaction object",
			"type", "itn",
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, errors.New("received nil interaction object")
	}

	if EnableVerboseLogging {
		logger.Debug("created new neuron interaction",
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
func (neuronInteraction *NeuronInteractionObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-neuronInteraction.resultsReadyChannel:
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
func (neuronInteraction *NeuronInteractionObject) GetFinalResults(timeout time.Duration) (result *api.NeuronInteractionResult, err error) {

	// Wait for the end of the interaction.
	err = neuronInteraction.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if neuronInteraction.finalResultsReceived {
		// If we received final results, verify the status.
		if neuronInteraction.FinalResultStatus == api.FinalResultStatus_FINAL_RESULT_STATUS_NEURON_RESULT {
			// Successful interaction, return result
			return neuronInteraction.finalResults, nil
		} else {
			// Unsuccessful interaction, return error
			errorString := fmt.Sprintf("%v: %v", neuronInteraction.FinalResultStatus, neuronInteraction.FinalStatus.Message)
			return nil, errors.New(errorString)
		}
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of neuron interaction")
	}
}
