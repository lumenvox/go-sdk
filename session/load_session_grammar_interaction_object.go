package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/status"
	"time"
)

// LoadSessionGrammarObject represents a session grammar load
type LoadSessionGrammarObject struct {
	correlationId string

	// Final result tracking
	loadComplete        bool
	finalResults        *api.SessionLoadGrammarResponse
	FinalStatus         *status.Status
	resultsReadyChannel chan struct{}
}

// prepareLoadSessionGrammar prepares the internal routing system
// for a LoadSessionGrammar response. It expects the correlationId of the
// relevant request, which is used to route the response.
func (session *SessionObject) prepareLoadSessionGrammar(correlationId string,
	loadSessionGrammarRequestObject *LoadSessionGrammarObject) (err error) {

	session.loadSessionGrammarMapLock.Lock()
	defer session.loadSessionGrammarMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.loadSessionGrammarMap[correlationId]; ok {
		return errors.New("load session grammar channel already exists")
	}

	// put a pointer to the request object in the map
	session.loadSessionGrammarMap[correlationId] = loadSessionGrammarRequestObject

	return
}

// NewInlineLoadSessionGrammar attempts to create a new load session grammar request for an inline grammar.
// If successful, a new request object will be returned.
func (session *SessionObject) NewInlineLoadSessionGrammar(language string,
	grammarLabel string,
	grammarText string,
	grammarSettings *api.GrammarSettings) (requestObject *LoadSessionGrammarObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create the object to track the request
	requestObject = &LoadSessionGrammarObject{
		correlationId:       correlationId,
		resultsReadyChannel: make(chan struct{}),
	}

	// Put a pointer to the object in a map (for response routing)
	err = session.prepareLoadSessionGrammar(correlationId, requestObject)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getInlineLoadSessionGrammarRequest(correlationId, language, grammarLabel,
		grammarText, grammarSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending LoadSessionGrammarRequest error: %v", err)
		logger.Error("sending load session grammar create request",
			"error", err.Error())
		return nil, err
	}

	if EnableVerboseLogging {
		logger.Debug("created new load session grammar request",
			"correlationId", requestObject.correlationId)
	}

	return requestObject, err
}

// NewUriLoadSessionGrammar attempts to create a new load session grammar request for a URI-based grammar.
// If successful, a new request object will be returned.
func (session *SessionObject) NewUriLoadSessionGrammar(language string,
	grammarLabel string,
	grammarUri string,
	grammarSettings *api.GrammarSettings) (requestObject *LoadSessionGrammarObject, err error) {

	logger := getLogger()

	// Generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// Create the object to track the request
	requestObject = &LoadSessionGrammarObject{
		correlationId:       correlationId,
		resultsReadyChannel: make(chan struct{}),
	}

	// Put a pointer to the object in a map (for response routing)
	err = session.prepareLoadSessionGrammar(correlationId, requestObject)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// send the request
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getUriLoadSessionGrammarRequest(correlationId, language, grammarLabel,
		grammarUri, grammarSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending LoadSessionGrammarRequest error: %v", err)
		logger.Error("sending load session grammar create request",
			"error", err.Error())
		return nil, err
	}

	if EnableVerboseLogging {
		logger.Debug("created new load session grammar request",
			"correlationId", requestObject.correlationId)
	}

	return requestObject, err
}

// WaitForFinalResults waits for the end of the request. This is typically
// triggered by final results, but it can also be triggered by errors.
//
// If nothing arrives before the timeout, an error will be returned. Note that
// request failures do not trigger errors from this function, so long as
// the notification arrives before the timeout.
func (loadSessionGrammarObject *LoadSessionGrammarObject) WaitForFinalResults(timeout time.Duration) error {

	select {
	case <-loadSessionGrammarObject.resultsReadyChannel:
		// resultsReadyChannel is closed when final results arrive
		return nil
	case <-time.After(timeout):
		return TimeoutError
	}
}

// GetFinalResults fetches the final results or error from the request,
// waiting if necessary. If the request succeeds, results will be returned.
//
// If the request does not complete before the specified timeout, an error will
// be returned.
func (loadSessionGrammarObject *LoadSessionGrammarObject) GetFinalResults(timeout time.Duration) (result *api.SessionLoadGrammarResponse, err error) {

	// Wait for the results
	err = loadSessionGrammarObject.WaitForFinalResults(timeout)
	if err != nil {
		return nil, err
	}

	if loadSessionGrammarObject.loadComplete {
		// Successful request, return result
		return loadSessionGrammarObject.finalResults, nil
	} else {
		// This should never happen.
		return nil, errors.New("unexpected end of load session grammar request")
	}
}
