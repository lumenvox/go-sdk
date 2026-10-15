package session

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lumenvox/protos-go/lumenvox/api"
)

// sessionResponseListener handles all responses from the specified session.
func sessionResponseListener(session *SessionObject, sessionIdChan chan string) {

	logger := getLogger()

	if EnableVerboseLogging {
		defer func() {
			logger.Debug("exiting responseHandler",
				"session", session.SessionId)
		}()
	}

	waitingOnSessionId := true
	var err error
	var response *api.SessionResponse

	for {
		response, err = session.sessionStream.Recv()
		if status.Code(err) == codes.Unauthenticated {
			// Token expiration detected. Handle it accordingly:
			logger.Error("sessionStream.Recv() failed with gRPC error: Unauthenticated",
				"error", err)
			if waitingOnSessionId {
				sessionIdChan <- ""
			}
			break
		}
		if err == io.EOF {
			if waitingOnSessionId {
				sessionIdChan <- ""
			}
			break
		}
		if err != nil {
			if statusErr, ok := status.FromError(err); ok {
				// Error type is status, so unpack it, for clearer reporting
				logger.Error("sessionStream.Recv() failed with status",
					"error", statusErr.Message())
				err = errors.New(statusErr.Message())
			} else {
				logger.Error("sessionStream.Recv() failed",
					"error", err)
			}

			if waitingOnSessionId {
				sessionIdChan <- ""
			}
			break
		}

		if EnableVerboseLogging {
			if response.GetAudioPull() == nil {
				logger.Debug("sessionStream.Recv() received",
					"response", response)
			} else {
				logger.Debug("sessionStream.Recv() received audio",
					"num_bytes", len(response.GetAudioPull().GetAudioData()),
					"correlation_id", response.CorrelationId.Value)
			}
		}

		if response.SessionId != nil && response.SessionId.Value != "" {
			if response.SessionId.Value != "" {
				if EnableVerboseLogging {
					logger.Debug("Recv response",
						"session_id", response.SessionId.Value)
				}
				if waitingOnSessionId {
					sessionIdChan <- response.SessionId.Value
					waitingOnSessionId = false
				} else {
					// if we weren't waiting on a session ID but got one anyway,
					// log a warning.
					logger.Warn("received extra session ID")
				}
			} else {
				logger.Error("Recv empty SessionId")
				if waitingOnSessionId {
					sessionIdChan <- ""
				}

				err = fmt.Errorf("empty SessionId")
				break
			}

			// The server may echo session_id on many stream messages. If a SessionEvent
			// is present (e.g. interaction create failure), it must still be routed;
			// otherwise only this branch runs and handleSessionEvent is never reached.
			if response.GetSessionEvent() != nil {
				if EnableVerboseLogging {
					logger.Debug("Recv session event",
						"response", response)
				}
				handleSessionEvent(session, response)
			}

		} else if response.GetVadEvent() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv VadEvent",
					"interaction_id", response.GetVadEvent().InteractionId,
					"type", response.GetVadEvent().VadEventType.String())
			}

			handleVadEvent(session, response)

		} else if response.GetPartialResult() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv PartialResult")
			}

			handlePartialResult(session, response)

		} else if response.GetFinalResult() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv FinalResult",
					"response", response)
			}

			handleFinalResult(session, response)

		} else if response.GetInteractionCreateAmd() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateAmdResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create AMD helper in the map
			session.interactionCreateAmdMapLock.Lock()
			createAmdHelper, ok := session.interactionCreateAmdMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateAmdMap, correlationId)
			session.interactionCreateAmdMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createAmdHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createAmdHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateAmd().InteractionId
			newInteractionObject := newAmdInteractionObject(interactionId)

			// add the new interaction object to the map
			session.amdInteractionsLock.Lock() // Protect concurrent map access
			session.amdInteractionsMap[interactionId] = newInteractionObject
			session.amdInteractionsLock.Unlock()

			// send the interaction object over the channel
			createAmdHelper.interactionCreateChannel <- &amdCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateAsr() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateAsrResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create ASR helper in the map
			session.interactionCreateAsrMapLock.Lock()
			createAsrHelper, ok := session.interactionCreateAsrMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateAsrMap, correlationId)
			session.interactionCreateAsrMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createAsrHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createAsrHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateAsr().InteractionId
			newInteractionObject := newAsrInteractionObject(interactionId)

			// add the new interaction object to the map
			session.asrInteractionsLock.Lock() // Protect concurrent map access
			session.asrInteractionsMap[interactionId] = newInteractionObject
			session.asrInteractionsLock.Unlock()

			// send the successful result over the channel
			result := &asrCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}
			createAsrHelper.interactionCreateChannel <- result

		} else if response.GetInteractionCreateCpa() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateCpaResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create CPA helper in the map
			session.interactionCreateCpaMapLock.Lock()
			createCpaHelper, ok := session.interactionCreateCpaMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateCpaMap, correlationId)
			session.interactionCreateCpaMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createCpaHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createCpaHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateCpa().InteractionId
			newInteractionObject := newCpaInteractionObject(interactionId)

			// add the new interaction object to the map
			session.cpaInteractionsLock.Lock() // Protect concurrent map access
			session.cpaInteractionsMap[interactionId] = newInteractionObject
			session.cpaInteractionsLock.Unlock()

			// send the interaction object over the channel
			createCpaHelper.interactionCreateChannel <- &cpaCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateDiarization() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateDiarizationResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create Diarization helper in the map
			session.interactionCreateDiarizationMapLock.Lock()
			createDiarizationHelper, ok := session.interactionCreateDiarizationMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateDiarizationMap, correlationId)
			session.interactionCreateDiarizationMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createDiarizationHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createDiarizationHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateDiarization().InteractionId
			newInteractionObject := &DiarizationInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				resultsReadyChannel:  make(chan struct{}),
			}

			// add the new interaction object to the map
			session.diarizationInteractionsLock.Lock() // Protect concurrent map access
			session.diarizationInteractionsMap[interactionId] = newInteractionObject
			session.diarizationInteractionsLock.Unlock()

			// send the interaction object over the channel
			createDiarizationHelper.interactionCreateChannel <- &diarizationCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateGrammarParse() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateGrammarParseResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create grammar parse helper in the map
			session.interactionCreateGrammarParseMapLock.Lock()
			createGrammarParseHelper, ok := session.interactionCreateGrammarParseMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateGrammarParseMap, correlationId)
			session.interactionCreateGrammarParseMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createGrammarParseHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createGrammarParseHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateGrammarParse().InteractionId
			newInteractionObject := &GrammarParseInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				FinalResultStatus:    api.FinalResultStatus_FINAL_RESULT_STATUS_UNSPECIFIED,
				resultsReadyChannel:  make(chan struct{}),
			}

			// add the new interaction object to the map
			session.grammarParseInteractionsLock.Lock() // Protect concurrent map access
			session.grammarParseInteractionsMap[interactionId] = newInteractionObject
			session.grammarParseInteractionsLock.Unlock()

			// send the interaction object over the channel
			createGrammarParseHelper.interactionCreateChannel <- &grammarParseCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateLanguageId() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateLanguageIdResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create LanguageId helper in the map
			session.interactionCreateLanguageIdMapLock.Lock()
			createLanguageIdHelper, ok := session.interactionCreateLanguageIdMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateLanguageIdMap, correlationId)
			session.interactionCreateLanguageIdMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createLanguageIdHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createLanguageIdHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateLanguageId().InteractionId
			newInteractionObject := &LanguageIdInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				resultsReadyChannel:  make(chan struct{}),
			}

			// add the new interaction object to the map
			session.languageIdInteractionsLock.Lock() // Protect concurrent map access
			session.languageIdInteractionsMap[interactionId] = newInteractionObject
			session.languageIdInteractionsLock.Unlock()

			// send the interaction object over the channel
			createLanguageIdHelper.interactionCreateChannel <- &languageIdCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateNlu() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateNluResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create NLU helper in the map
			session.interactionCreateNluMapLock.Lock()
			createNluHelper, ok := session.interactionCreateNluMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateNluMap, correlationId)
			session.interactionCreateNluMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createNluHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createNluHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateNlu().InteractionId
			newInteractionObject := &NluInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				FinalResultStatus:    api.FinalResultStatus_FINAL_RESULT_STATUS_UNSPECIFIED,
				resultsReadyChannel:  make(chan struct{}),
			}

			// add the new interaction object to the map
			session.nluInteractionsLock.Lock() // Protect concurrent map access
			session.nluInteractionsMap[interactionId] = newInteractionObject
			session.nluInteractionsLock.Unlock()

			// send the interaction object over the channel
			createNluHelper.interactionCreateChannel <- &nluCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateNormalizeText() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateNormalizeTextResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create normalization helper in the map
			session.interactionCreateNormalizeTextMapLock.Lock()
			createNormalizationHelper, ok := session.interactionCreateNormalizeTextMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateNormalizeTextMap, correlationId)
			session.interactionCreateNormalizeTextMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createNormalizationHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createNormalizationHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateNormalizeText().InteractionId
			newInteractionObject := &NormalizationInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				FinalResultStatus:    api.FinalResultStatus_FINAL_RESULT_STATUS_UNSPECIFIED,
				resultsReadyChannel:  make(chan struct{}),
			}

			// add the new interaction object to the map
			session.normalizationInteractionsLock.Lock() // Protect concurrent map access
			session.normalizationInteractionsMap[interactionId] = newInteractionObject
			session.normalizationInteractionsLock.Unlock()

			// send the interaction object over the channel
			createNormalizationHelper.interactionCreateChannel <- &normalizationCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateNeuron() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateNeuronResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create neuron helper in the map
			session.interactionCreateNeuronMapLock.Lock()
			createNeuronHelper, ok := session.interactionCreateNeuronMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateNeuronMap, correlationId)
			session.interactionCreateNeuronMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createNeuronHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createNeuronHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateNeuron().InteractionId
			newInteractionObject := &NeuronInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				FinalResultStatus:    api.FinalResultStatus_FINAL_RESULT_STATUS_UNSPECIFIED,
				resultsReadyChannel:  make(chan struct{}),
			}

			// add the new interaction object to the map
			session.neuronInteractionsLock.Lock() // Protect concurrent map access
			session.neuronInteractionsMap[interactionId] = newInteractionObject
			session.neuronInteractionsLock.Unlock()

			// send the successful result over the channel
			createNeuronHelper.interactionCreateChannel <- &neuronCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateSentiment() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateSentimentResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create sentiment helper in the map
			session.interactionCreateSentimentMapLock.Lock()
			createSentimentHelper, ok := session.interactionCreateSentimentMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateSentimentMap, correlationId)
			session.interactionCreateSentimentMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createSentimentHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createSentimentHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateSentiment().InteractionId
			newInteractionObject := &SentimentInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				FinalResultStatus:    api.FinalResultStatus_FINAL_RESULT_STATUS_UNSPECIFIED,
				resultsReadyChannel:  make(chan struct{}),
			}

			// add the new interaction object to the map
			session.sentimentInteractionsLock.Lock() // Protect concurrent map access
			session.sentimentInteractionsMap[interactionId] = newInteractionObject
			session.sentimentInteractionsLock.Unlock()

			// send the interaction object over the channel
			createSentimentHelper.interactionCreateChannel <- &sentimentCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetInteractionCreateTranscription() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateTranscriptionResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create transcription helper in the map
			session.interactionCreateTranscriptionMapLock.Lock()
			createTranscriptionHelper, ok := session.interactionCreateTranscriptionMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateTranscriptionMap, correlationId)
			session.interactionCreateTranscriptionMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createTranscriptionHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createTranscriptionHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateTranscription().InteractionId
			newInteractionObject := &TranscriptionInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				resultsReadyChannel:  make(chan struct{}),
				vadCurrentState:      api.VadEvent_VAD_EVENT_TYPE_UNSPECIFIED,

				isContinuousTranscription: createTranscriptionHelper.isContinuousTranscription,
			}
			newInteractionObject.partialResultsChannels = append(newInteractionObject.partialResultsChannels, make(chan struct{}))
			newInteractionObject.vadBeginProcessingChannels = append(newInteractionObject.vadBeginProcessingChannels, make(chan struct{}))
			newInteractionObject.vadBargeInChannels = append(newInteractionObject.vadBargeInChannels, make(chan struct{}))
			newInteractionObject.vadBargeOutChannels = append(newInteractionObject.vadBargeOutChannels, make(chan struct{}))
			newInteractionObject.vadBargeInTimeoutChannels = append(newInteractionObject.vadBargeInTimeoutChannels, make(chan struct{}))
			newInteractionObject.vadBargeOutTimeoutChannels = append(newInteractionObject.vadBargeOutTimeoutChannels, make(chan struct{}))
			newInteractionObject.vadRecordLog = append(newInteractionObject.vadRecordLog, createEmptyVadInteractionRecord())

			// add the new interaction object to the map
			session.transcriptionInteractionsLock.Lock() // Protect concurrent map access
			session.transcriptionInteractionsMap[interactionId] = newInteractionObject
			session.transcriptionInteractionsLock.Unlock()

			// send the successful result over the channel
			result := &transcriptionCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}
			createTranscriptionHelper.interactionCreateChannel <- result

		} else if response.GetInteractionCreateTts() != nil {

			if EnableVerboseLogging {
				logger.Debug("recv interaction create response",
					"type", "InteractionCreateTtsResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the interaction create TTS helper in the map
			session.interactionCreateTtsMapLock.Lock()
			createTtsHelper, ok := session.interactionCreateTtsMap[correlationId]
			// each helper should only be used once, and we already have the helper,
			// so delete the map entry before unlocking.
			delete(session.interactionCreateTtsMap, correlationId)
			session.interactionCreateTtsMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing interaction create",
					"error", "interaction create helper does not exist",
					"response", response)
				continue
			}

			// catch nil helper
			if createTtsHelper == nil {
				logger.Warn("error routing interaction create",
					"error", "interaction create channel is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(createTtsHelper.deadline) {
				logger.Warn("error routing interaction create",
					"error", "interaction create timed out",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. create the interaction object
			interactionId := response.GetInteractionCreateTts().InteractionId
			newInteractionObject := &TtsInteractionObject{
				InteractionId:        interactionId,
				finalResultsReceived: false,
				finalResults:         nil,
				FinalResultStatus:    api.FinalResultStatus_FINAL_RESULT_STATUS_UNSPECIFIED,
				resultsReadyChannel:  make(chan struct{}),
			}
			newInteractionObject.partialResultsChannels = append(newInteractionObject.partialResultsChannels, make(chan struct{}))

			// add the new interaction object to the map
			session.ttsInteractionsLock.Lock() // Protect concurrent map access
			session.ttsInteractionsMap[interactionId] = newInteractionObject
			session.ttsInteractionsLock.Unlock()

			// send the interaction object over the channel
			createTtsHelper.interactionCreateChannel <- &ttsCreateResult{
				InteractionObject: newInteractionObject,
				Error:             nil,
			}

		} else if response.GetAudioPull() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv GetAudioPull",
					"num_bytes", len(response.GetAudioPull().GetAudioData()))
			}

			handleAudioPull(session, response)

		} else if result := response.GetSessionGrammar(); result != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv SessionLoadGrammarResponse",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing session grammar",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing session grammar",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the request object in the map
			session.loadSessionGrammarMapLock.Lock()
			requestObject, ok := session.loadSessionGrammarMap[correlationId]
			// there should only be one response per request, so delete the map
			// entry before unlocking.
			delete(session.loadSessionGrammarMap, correlationId)
			session.loadSessionGrammarMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing session grammar",
					"error", "request object does not exist",
					"response", response)
				continue
			}

			// catch nil object
			if requestObject == nil {
				logger.Warn("error routing session grammar",
					"error", "request object is nil",
					"response", response)
				continue
			}

			// if we got this far, the response should be OK. update the request object
			requestObject.loadComplete = true
			requestObject.finalResults = result
			requestObject.FinalStatus = result.Status
			close(requestObject.resultsReadyChannel)

		} else if response.GetSessionGetSettings() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv SessionGetSettings",
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing session settings",
					"error", "missing correlation_id",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing session settings",
					"error", "empty correlation_id",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// look for the request object in the map
			session.sessionGetSettingsMapLock.Lock()
			requestObject, ok := session.sessionGetSettingsMap[correlationId]
			// there should only be one response per request, so delete the map
			// entry before unlocking.
			delete(session.sessionGetSettingsMap, correlationId)
			session.sessionGetSettingsMapLock.Unlock()

			// catch missing map entry
			if ok == false {
				logger.Warn("error routing session settings",
					"error", "request object does not exist",
					"response", response)
				continue
			}

			// catch nil object
			if requestObject == nil {
				logger.Warn("error routing session settings",
					"error", "request object is nil",
					"response", response)
				continue
			}

			// catch timeout
			if time.Now().After(requestObject.deadline) {
				logger.Warn("error routing session settings",
					"error", "session settings timed out",
					"response", response)
				continue
			}

			// send the successful result over the channel
			result := &sessionSettingsResult{
				SessionSettings: response.GetSessionGetSettings(),
				Error:           nil,
			}
			requestObject.sessionSettingsChannel <- result

		} else {

			if response.GetSessionClose() != nil {

				if EnableVerboseLogging {
					logger.Debug("Recv SessionCloseResponse",
						"response", response)
				}
				err = nil
				break

			} else if response.GetSessionEvent() != nil {

				if EnableVerboseLogging {
					logger.Debug("Recv session event",
						"response", response)
				}

				handleSessionEvent(session, response)

			} else {
				logger.Error("Recv unexpected response type",
					"response", response)
			}
		}
	}

	// Signal used by SessionClose method to wait for SessionClose response before closing the stream.
	session.sessionCloseChannel <- struct{}{}

	// Signal to session object here that we're out of the listening loop
	// and set the reason that loop exited (could be nil if no error)
	session.StreamLoopExitErr.Store(&err)
}

func handleVadEvent(session *SessionObject, response *api.SessionResponse) {

	logger := getLogger()

	// Get the interaction id, to find the interaction object.
	interactionId := response.GetVadEvent().GetInteractionId()

	// Try ASR interactions first
	session.asrInteractionsLock.RLock()
	if interactionObject, ok := session.asrInteractionsMap[interactionId]; ok {
		session.asrInteractionsLock.RUnlock()

		// This is an ASR interaction.
		interactionObject.handleVadEvent(response.GetVadEvent())
		return
	}
	session.asrInteractionsLock.RUnlock()

	// Try transcription interactions
	session.transcriptionInteractionsLock.RLock()
	if interactionObject, ok := session.transcriptionInteractionsMap[interactionId]; ok {
		session.transcriptionInteractionsLock.RUnlock()

		// This is a transcription interaction.
		interactionObject.vadEventsLock.Lock()
		// Get the index of the current VAD interaction. This should only ever be greater
		// than 0 when we're dealing with a continuous transcription.
		currentVadInteractionIndex := interactionObject.vadInteractionCounter
		// Get the last received state for the current VAD interaction.
		currentVadInteractionState := interactionObject.vadCurrentState

		// Based on the current vad state, update the interaction.
		switch currentVadInteractionState {

		case api.VadEvent_VAD_EVENT_TYPE_UNSPECIFIED:
			// this is our first VAD event. Based on the type, update the interaction.
			switch response.GetVadEvent().VadEventType {

			case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
				// we got a BEGIN_PROCESSING event. This is an expected transition, so
				// no need to do anything special.

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING
				// Update the VAD event log
				interactionObject.vadRecordLog[currentVadInteractionIndex].beginProcessingReceived = true
				// To provide for functionality like "WaitForNextBeginProcessing", create
				// the next set of VAD event channels and a new VAD record
				interactionObject.vadBeginProcessingChannels = append(interactionObject.vadBeginProcessingChannels, make(chan struct{}))
				interactionObject.vadBargeInChannels = append(interactionObject.vadBargeInChannels, make(chan struct{}))
				interactionObject.vadBargeOutChannels = append(interactionObject.vadBargeOutChannels, make(chan struct{}))
				interactionObject.vadBargeInTimeoutChannels = append(interactionObject.vadBargeInTimeoutChannels, make(chan struct{}))
				interactionObject.vadBargeOutTimeoutChannels = append(interactionObject.vadBargeOutTimeoutChannels, make(chan struct{}))
				interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())
				// Now that the event has been recorded, signal the channel.
				close(interactionObject.vadBeginProcessingChannels[currentVadInteractionIndex])
			default:
				// We got something other than a BEGIN_PROCESSING event. For now, the SDK
				// expects a BEGIN_PROCESSING event to start every new VAD interaction, so
				// we have received an invalid transition. Log a warning.
				logger.Warn("received VAD event before BEGIN_PROCESSING",
					"response", response.GetVadEvent().VadEventType.String())
			}

		case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
			// we have already received a begin_processing event. Based on the type of the
			// new event, update the interaction.
			switch response.GetVadEvent().VadEventType {

			case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN:
				// we got a BARGE_IN event. This is an expected transition, so no need
				// to do anything special.

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_BARGE_IN
				// Update the VAD event log
				bargeInAudioOffset := response.GetVadEvent().AudioOffset
				var bargeInTime int
				if bargeInAudioOffset == nil {
					bargeInTime = 1
					logger.Warn("VAD BARGE-IN event detected with empty AudioOffset. Using 1.")
				} else {
					bargeInTime = int(bargeInAudioOffset.Value)
				}
				interactionObject.vadRecordLog[currentVadInteractionIndex].bargeInReceived = bargeInTime
				// Now that the event has been recorded, signal the channel.
				close(interactionObject.vadBargeInChannels[currentVadInteractionIndex])

			case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN_TIMEOUT:
				// we got a BARGE_IN_TIMEOUT event. This is an expected transition, so no
				// need to do anything special.

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_BARGE_IN_TIMEOUT
				// Update the VAD event log
				interactionObject.vadRecordLog[currentVadInteractionIndex].bargeInTimeoutReceived = true
				// Increment the vad interaction counter
				interactionObject.vadInteractionCounter++
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBargeInTimeoutChannels[currentVadInteractionIndex])
				// Update the local counter
				currentVadInteractionIndex = interactionObject.vadInteractionCounter

			default:
				// We received something that doesn't make sense after a BEGIN_PROCESSING.
				// In other words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"from_state", currentVadInteractionState.String(),
					"to_state", response.GetVadEvent().VadEventType.String())
			}

		case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN:
			// we have already received a barge_in event. Based on the type of the new event,
			// update the interaction.
			switch response.GetVadEvent().VadEventType {

			case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH:
				// we got an END_OF_SPEECH event. This is an expected transition, so no
				// need to do anything special

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH
				// Update the VAD event log
				bargeOutAudioOffset := response.GetVadEvent().AudioOffset
				var bargeOutTime int
				if bargeOutAudioOffset == nil {
					bargeOutTime = 1
					logger.Warn("VAD END_OF_SPEECH event detected with empty AudioOffset. Using 1.")
				} else {
					bargeOutTime = int(bargeOutAudioOffset.Value)
				}
				interactionObject.vadRecordLog[currentVadInteractionIndex].bargeOutReceived = bargeOutTime
				// Increment the vad interaction counter
				interactionObject.vadInteractionCounter++
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBargeOutChannels[currentVadInteractionIndex])
				// Update the local counter
				currentVadInteractionIndex = interactionObject.vadInteractionCounter

			case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH_TIMEOUT:
				// we got an END_OF_SPEECH_TIMEOUT event. This is a valid transition, but it is also
				// an error.

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH_TIMEOUT
				// Update the VAD event log
				interactionObject.vadRecordLog[currentVadInteractionIndex].bargeInTimeoutReceived = true
				// Increment the vad interaction counter
				interactionObject.vadInteractionCounter++
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBargeOutTimeoutChannels[currentVadInteractionIndex])
				// Update the local counter
				currentVadInteractionIndex = interactionObject.vadInteractionCounter

			// TODO: support more VAD cases here
			default:
				// We received something that doesn't make sense after a BARGE_IN. In other
				// words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"from_state", currentVadInteractionState.String(),
					"to_state", response.GetVadEvent().VadEventType.String())
			}

		case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH:
			// We have already received an END_OF_SPEECH event. Based on the type of the
			// new event, update the interaction.
			switch response.GetVadEvent().VadEventType {

			case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
				// we got a BEGIN_PROCESSING event. This is an expected transition for
				// continuous interactions.

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING
				// Update the VAD event log
				interactionObject.vadRecordLog[currentVadInteractionIndex].beginProcessingReceived = true
				// To provide for functionality like "WaitForNextBeginProcessing", create
				// the next set of VAD event channels and a VAD event log.
				interactionObject.vadBeginProcessingChannels = append(interactionObject.vadBeginProcessingChannels, make(chan struct{}))
				interactionObject.vadBargeInChannels = append(interactionObject.vadBargeInChannels, make(chan struct{}))
				interactionObject.vadBargeOutChannels = append(interactionObject.vadBargeOutChannels, make(chan struct{}))
				interactionObject.vadBargeInTimeoutChannels = append(interactionObject.vadBargeInTimeoutChannels, make(chan struct{}))
				interactionObject.vadBargeOutTimeoutChannels = append(interactionObject.vadBargeOutTimeoutChannels, make(chan struct{}))
				interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBeginProcessingChannels[currentVadInteractionIndex])

			default:
				// We received something that doesn't make sense after an END_OF_SPEECH. In
				// other words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"from_state", currentVadInteractionState.String(),
					"to_state", response.GetVadEvent().VadEventType.String())
			}

		case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN_TIMEOUT:
			// We have already received a BARGE_IN_TIMEOUT event. Based on the type of the
			// new event, update the interaction.
			switch response.GetVadEvent().VadEventType {

			case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
				// we got a BEGIN_PROCESSING event. This is a valid transition for
				// continuous interactions.

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING
				// Update the VAD event log
				interactionObject.vadRecordLog[currentVadInteractionIndex].beginProcessingReceived = true
				// To provide for functionality like "WaitForNextBeginProcessing", create
				// the next set of VAD event channels.
				interactionObject.vadBeginProcessingChannels = append(interactionObject.vadBeginProcessingChannels, make(chan struct{}))
				interactionObject.vadBargeInChannels = append(interactionObject.vadBargeInChannels, make(chan struct{}))
				interactionObject.vadBargeOutChannels = append(interactionObject.vadBargeOutChannels, make(chan struct{}))
				interactionObject.vadBargeInTimeoutChannels = append(interactionObject.vadBargeInTimeoutChannels, make(chan struct{}))
				interactionObject.vadBargeOutTimeoutChannels = append(interactionObject.vadBargeOutTimeoutChannels, make(chan struct{}))
				interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBeginProcessingChannels[currentVadInteractionIndex])

			default:
				// We received something that doesn't make sense after a BARGE_IN_TIMEOUT. In
				// other words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"from_state", currentVadInteractionState.String(),
					"to_state", response.GetVadEvent().VadEventType.String())
			}

		case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH_TIMEOUT:
			// We previously received an END_OF_SPEECH_TIMEOUT event. Based on the type of the
			// new event, update the interaction.
			//
			// Note that interactions are not expected to proceed past an end-of-speech timeout,
			// so this code will likely be unused.
			switch response.GetVadEvent().VadEventType {

			case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
				// we got a BEGIN_PROCESSING event. This is a valid transition for
				// continuous interactions.

				// Update the current state
				interactionObject.vadCurrentState = api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING
				// Update the VAD event log
				interactionObject.vadRecordLog[currentVadInteractionIndex].beginProcessingReceived = true
				// To provide for functionality like "WaitForNextBeginProcessing", create
				// the next set of VAD event channels.
				interactionObject.vadBeginProcessingChannels = append(interactionObject.vadBeginProcessingChannels, make(chan struct{}))
				interactionObject.vadBargeInChannels = append(interactionObject.vadBargeInChannels, make(chan struct{}))
				interactionObject.vadBargeOutChannels = append(interactionObject.vadBargeOutChannels, make(chan struct{}))
				interactionObject.vadBargeInTimeoutChannels = append(interactionObject.vadBargeInTimeoutChannels, make(chan struct{}))
				interactionObject.vadBargeOutTimeoutChannels = append(interactionObject.vadBargeOutTimeoutChannels, make(chan struct{}))
				interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBeginProcessingChannels[currentVadInteractionIndex])

			default:
				// We received something that doesn't make sense after an END_OF_SPEECH_TIMEOUT.
				// In other words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"from_state", currentVadInteractionState.String(),
					"to_state", response.GetVadEvent().VadEventType.String())
			}

		default:
			// Unexpected current state. Should never happen, but just in case, log a warning
			logger.Warn("unexpected VAD",
				"current_state", currentVadInteractionState.String())
		}
		interactionObject.vadEventsLock.Unlock()
		return
	}
	session.transcriptionInteractionsLock.RUnlock()

	// Try AMD interactions
	session.amdInteractionsLock.RLock()
	if interactionObject, ok := session.amdInteractionsMap[interactionId]; ok {
		session.amdInteractionsLock.RUnlock()

		// This is an AMD interaction.
		interactionObject.handleVadEvent(response.GetVadEvent())
		return
	}
	session.amdInteractionsLock.RUnlock()

	// Try CPA interactions
	session.cpaInteractionsLock.RLock()
	if interactionObject, ok := session.cpaInteractionsMap[interactionId]; ok {
		session.cpaInteractionsLock.RUnlock()

		// This is a CPA interaction.
		interactionObject.handleVadEvent(response.GetVadEvent())
		return
	}
	session.cpaInteractionsLock.RUnlock()

	// We did not find the interaction.
	logger.Error("Recv VadEvent: interaction not found",
		"interaction_id", interactionId)
}

func handleFinalResult(session *SessionObject, response *api.SessionResponse) {

	logger := getLogger()

	// Get the interaction id, to find the interaction object.
	interactionId := response.GetFinalResult().GetInteractionId()
	finalResult := response.GetFinalResult()
	if finalResult == nil {
		logger.Warn("handleFinalResult received nil FinalResult",
			"interaction_id", interactionId)
		return
	}

	// Try ASR interactions first
	session.asrInteractionsLock.Lock()
	if interactionObject, interactionFound := session.asrInteractionsMap[interactionId]; interactionFound {
		// This is an ASR interaction.
		// We support grammar-based alternatives for multiple interaction types, so we need to figure
		// out which was active here.
		if finalResult.GetFinalResult().GetCpaInteractionResult() != nil {
			newAsrInteractionResult := &api.AsrInteractionResult{
				NBests: []*api.AsrGrammarResult{finalResult.GetFinalResult().GetCpaInteractionResult().CpaResult},
			}
			interactionObject.finalResults = newAsrInteractionResult
		} else if finalResult.GetFinalResult().GetAmdInteractionResult() != nil {
			newAsrInteractionResult := &api.AsrInteractionResult{
				NBests: []*api.AsrGrammarResult{finalResult.GetFinalResult().GetAmdInteractionResult().AmdResult},
			}
			interactionObject.finalResults = newAsrInteractionResult
		} else {
			// If we didn't detect CPA or AMD interaction results, assume a plain ASR interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetAsrInteractionResult()
		}
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.asrInteractionsMap, interactionId)
		session.asrInteractionsLock.Unlock()
		return
	}
	session.asrInteractionsLock.Unlock()

	// Try transcription interactions
	session.transcriptionInteractionsLock.Lock()
	if interactionObject, interactionFound := session.transcriptionInteractionsMap[interactionId]; interactionFound {
		// This is a transcription interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetTranscriptionInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.transcriptionInteractionsMap, interactionId)
		session.transcriptionInteractionsLock.Unlock()
		return
	}
	session.transcriptionInteractionsLock.Unlock()

	// Try NLU interactions
	session.nluInteractionsLock.Lock()
	if interactionObject, interactionFound := session.nluInteractionsMap[interactionId]; interactionFound {
		// This is an NLU interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetNluInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.nluInteractionsMap, interactionId)
		session.nluInteractionsLock.Unlock()
		return
	}
	session.nluInteractionsLock.Unlock()

	// Try grammar parse interactions
	session.grammarParseInteractionsLock.Lock()
	if interactionObject, interactionFound := session.grammarParseInteractionsMap[interactionId]; interactionFound {
		// This is a grammar parse interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetGrammarParseInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.grammarParseInteractionsMap, interactionId)
		session.grammarParseInteractionsLock.Unlock()
		return
	}
	session.grammarParseInteractionsLock.Unlock()

	// Try AMD interactions
	session.amdInteractionsLock.Lock()
	if interactionObject, interactionFound := session.amdInteractionsMap[interactionId]; interactionFound {
		// This is an AMD interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetAmdInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.amdInteractionsMap, interactionId)

		session.amdInteractionsLock.Unlock()
		return
	}
	session.amdInteractionsLock.Unlock()

	// Try CPA interactions
	session.cpaInteractionsLock.Lock()
	if interactionObject, interactionFound := session.cpaInteractionsMap[interactionId]; interactionFound {
		// This is a CPA interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetCpaInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.cpaInteractionsMap, interactionId)
		session.cpaInteractionsLock.Unlock()
		return
	}
	session.cpaInteractionsLock.Unlock()

	// Try normalization interactions
	session.normalizationInteractionsLock.Lock()
	if interactionObject, interactionFound := session.normalizationInteractionsMap[interactionId]; interactionFound {
		// This is a normalization interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetNormalizeTextResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.normalizationInteractionsMap, interactionId)
		session.normalizationInteractionsLock.Unlock()
		return
	}
	session.normalizationInteractionsLock.Unlock()

	// Try neuron interactions
	session.neuronInteractionsLock.Lock()
	if interactionObject, interactionFound := session.neuronInteractionsMap[interactionId]; interactionFound {
		// This is a neuron interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetNeuronInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.neuronInteractionsMap, interactionId)

		session.neuronInteractionsLock.Unlock()
		return
	}
	session.neuronInteractionsLock.Unlock()

	// Try sentiment interactions
	session.sentimentInteractionsLock.Lock()
	if interactionObject, interactionFound := session.sentimentInteractionsMap[interactionId]; interactionFound {
		// This is a sentiment interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetSentimentInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.sentimentInteractionsMap, interactionId)
		session.sentimentInteractionsLock.Unlock()
		return
	}
	session.sentimentInteractionsLock.Unlock()

	// Try TTS interactions
	session.ttsInteractionsLock.Lock()
	if interactionObject, interactionFound := session.ttsInteractionsMap[interactionId]; interactionFound {
		// This is a TTS interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetTtsInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.ttsInteractionsMap, interactionId)
		session.ttsInteractionsLock.Unlock()
		return
	}
	session.ttsInteractionsLock.Unlock()

	// Try diarization interactions
	session.diarizationInteractionsLock.Lock()
	if interactionObject, interactionFound := session.diarizationInteractionsMap[interactionId]; interactionFound {
		// This is a diarization interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetDiarizationInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.diarizationInteractionsMap, interactionId)

		session.diarizationInteractionsLock.Unlock()
		return
	}
	session.diarizationInteractionsLock.Unlock()

	// Try language ID interactions
	session.languageIdInteractionsLock.Lock()
	if interactionObject, interactionFound := session.languageIdInteractionsMap[interactionId]; interactionFound {
		// This is a language ID interaction.
		interactionObject.finalResults = finalResult.GetFinalResult().GetLanguageIdInteractionResult()
		interactionObject.FinalStatus = finalResult.Status
		interactionObject.FinalResultStatus = finalResult.FinalResultStatus
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)
		delete(session.languageIdInteractionsMap, interactionId)
		session.languageIdInteractionsLock.Unlock()
		return
	}
	session.languageIdInteractionsLock.Unlock()

	// We did not find the interaction.
	logger.Error("Recv FinalResult: interaction not found",
		"interaction_id", interactionId)
}

func handlePartialResult(session *SessionObject, response *api.SessionResponse) {

	logger := getLogger()

	// Get the interaction id, to find the interaction object.
	interactionId := response.GetPartialResult().GetInteractionId()

	// Try ASR interactions first
	session.asrInteractionsLock.RLock()
	if interactionObject, ok := session.asrInteractionsMap[interactionId]; ok {
		session.asrInteractionsLock.RUnlock()

		// This is an ASR interaction.

		// Synchronize tracking information.
		interactionObject.partialResultLock.Lock()
		// Get the index of this partial result.
		partialResultIdx := interactionObject.partialResultsReceived
		// Store the new partial result.
		interactionObject.partialResultsList = append(interactionObject.partialResultsList, response.GetPartialResult())
		// Create a channel for the next partial result.
		nextPartialChannel := make(chan struct{})
		interactionObject.partialResultsChannels = append(interactionObject.partialResultsChannels, nextPartialChannel)
		// Update the number of partial results received.
		interactionObject.partialResultsReceived = partialResultIdx + 1
		// Close the channel for this partial result.
		close(interactionObject.partialResultsChannels[partialResultIdx])
		// Done with updates; release lock.
		interactionObject.partialResultLock.Unlock()
		return
	}
	session.asrInteractionsLock.RUnlock()

	// Try transcription interactions
	session.transcriptionInteractionsLock.RLock()
	if interactionObject, ok := session.transcriptionInteractionsMap[interactionId]; ok {
		session.transcriptionInteractionsLock.RUnlock()

		// This is a transcription interaction.

		// Synchronize tracking information.
		interactionObject.partialResultLock.Lock()
		// Get the index of this partial result.
		partialResultIdx := interactionObject.partialResultsReceived
		// Store the new partial result.
		interactionObject.partialResultsList = append(interactionObject.partialResultsList, response.GetPartialResult())
		// Create a channel for the next partial result.
		nextPartialChannel := make(chan struct{})
		interactionObject.partialResultsChannels = append(interactionObject.partialResultsChannels, nextPartialChannel)
		// Update the number of partial results received.
		interactionObject.partialResultsReceived = partialResultIdx + 1
		// Close the channel for this partial result.
		close(interactionObject.partialResultsChannels[partialResultIdx])
		// Done with updates; release lock.
		interactionObject.partialResultLock.Unlock()
		return
	}
	session.transcriptionInteractionsLock.RUnlock()

	// Try TTS interactions
	session.ttsInteractionsLock.RLock()
	if interactionObject, ok := session.ttsInteractionsMap[interactionId]; ok {
		session.ttsInteractionsLock.RUnlock()

		// This is a TTS interaction.

		// Synchronize tracking information.
		interactionObject.partialResultLock.Lock()
		// Get the index of this partial result.
		partialResultIdx := interactionObject.partialResultsReceived
		// Store the new partial result.
		interactionObject.partialResultsList = append(interactionObject.partialResultsList, response.GetPartialResult())
		// Create a channel for the next partial result.
		nextPartialChannel := make(chan struct{})
		interactionObject.partialResultsChannels = append(interactionObject.partialResultsChannels, nextPartialChannel)
		// Update the number of partial results received.
		interactionObject.partialResultsReceived = partialResultIdx + 1
		// Close the channel for this partial result.
		close(interactionObject.partialResultsChannels[partialResultIdx])
		// Done with updates; release lock.
		interactionObject.partialResultLock.Unlock()
		return
	}
	session.ttsInteractionsLock.RUnlock()

	// We did not find the interaction.
	logger.Error("Recv PartialResult: interaction not found",
		"interaction_id", interactionId)
}

func handleSessionEvent(session *SessionObject, response *api.SessionResponse) {
	// Check if this SessionEvent has a correlation ID for routing to specific handlers
	if response.CorrelationId != nil && response.CorrelationId.Value != "" {
		correlationId := response.CorrelationId.Value

		// Check for interaction create ASR error response
		if handleInteractionCreateAsrError(session, response, correlationId) {
			return
		}

		// Check for interaction create transcription error response
		if handleInteractionCreateTranscriptionError(session, response, correlationId) {
			return
		}

		// Check for interaction create neuron error response
		if handleInteractionCreateNeuronError(session, response, correlationId) {
			return
		}

		// Check for interaction create AMD error response
		if handleInteractionCreateAmdError(session, response, correlationId) {
			return
		}

		// Check for interaction create CPA error response
		if handleInteractionCreateCpaError(session, response, correlationId) {
			return
		}

		// Check for interaction create diarization error response
		if handleInteractionCreateDiarizationError(session, response, correlationId) {
			return
		}

		// Check for interaction create grammar parse error response
		if handleInteractionCreateGrammarParseError(session, response, correlationId) {
			return
		}

		// Check for interaction create language id error response
		if handleInteractionCreateLanguageIdError(session, response, correlationId) {
			return
		}

		// Check for interaction create NLU error response
		if handleInteractionCreateNluError(session, response, correlationId) {
			return
		}

		// Check for interaction create normalization error response
		if handleInteractionCreateNormalizationError(session, response, correlationId) {
			return
		}

		// Check for interaction create sentiment error response
		if handleInteractionCreateSentimentError(session, response, correlationId) {
			return
		}

		// Check for interaction create TTS error response
		if handleInteractionCreateTtsError(session, response, correlationId) {
			return
		}

		// Check for finalize interaction response
		if handleFinalizeInteractionResponse(session, response, correlationId) {
			return
		}

		// Check for session set settings response
		if handleSessionSetSettingsResponse(session, response, correlationId) {
			return
		}

		// Check for audio pull error response
		if handleAudioPullError(session, response, correlationId) {
			return
		}

		// Add other correlation-based handlers here as needed
	}

	// Handle other SessionEvent types (grammar errors, etc.)
	handleOtherSessionEvents(session, response)
}

// interactionCreateError builds the error returned to a caller whose interaction
// create request failed. The server's status (code, message, and details) is
// preserved so callers can inspect it via status.FromError.
func interactionCreateError(response *api.SessionResponse, interactionType string, correlationId string) error {
	se := response.GetSessionEvent()
	if se == nil || se.StatusMessage == nil {
		return status.Errorf(codes.Internal, "%s interaction creation failed: unknown error", interactionType)
	}

	sm := se.StatusMessage
	getLogger().Error("interaction create "+interactionType+" failed",
		"error", sm.Message,
		"code", sm.Code,
		"correlation_id", correlationId)

	if codes.Code(sm.Code) == codes.OK {
		// status.ErrorProto returns nil for OK, so report it as Internal instead.
		// Proto() returns a copy, leaving the response itself unchanged.
		st := status.FromProto(sm).Proto()
		st.Code = int32(codes.Internal)
		return status.ErrorProto(st)
	}
	return status.ErrorProto(sm)
}

func handleInteractionCreateAsrError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateAsrMapLock.Lock()
	createAsrHelper, exists := session.interactionCreateAsrMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateAsrMap, correlationId)
	}
	session.interactionCreateAsrMapLock.Unlock()

	if !exists {
		return false // not an ASR create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create ASR error response",
			"response", response)
	}

	// catch nil helper
	if createAsrHelper == nil {
		logger.Warn("error routing interaction create ASR response",
			"error", "ASR helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createAsrHelper.deadline) {
		logger.Warn("error routing interaction create ASR response",
			"error", "interaction create ASR timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "ASR", correlationId)

	// Send the error result through the channel
	result := &asrCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	createAsrHelper.interactionCreateChannel <- result
	return true
}

func handleInteractionCreateTranscriptionError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateTranscriptionMapLock.Lock()
	createTranscriptionHelper, exists := session.interactionCreateTranscriptionMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateTranscriptionMap, correlationId)
	}
	session.interactionCreateTranscriptionMapLock.Unlock()

	if !exists {
		return false // not a transcription create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create transcription error response",
			"response", response)
	}

	// catch nil helper
	if createTranscriptionHelper == nil {
		logger.Warn("error routing interaction create transcription response",
			"error", "transcription helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createTranscriptionHelper.deadline) {
		logger.Warn("error routing interaction create transcription response",
			"error", "interaction create transcription timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "transcription", correlationId)

	// Send the error result through the channel
	result := &transcriptionCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	createTranscriptionHelper.interactionCreateChannel <- result
	return true
}

func handleInteractionCreateNeuronError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateNeuronMapLock.Lock()
	createNeuronHelper, exists := session.interactionCreateNeuronMap[correlationId]
	if exists {
		delete(session.interactionCreateNeuronMap, correlationId)
	}
	session.interactionCreateNeuronMapLock.Unlock()

	if !exists {
		return false
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create neuron error response",
			"response", response)
	}

	if createNeuronHelper == nil {
		logger.Warn("error routing interaction create neuron response",
			"error", "neuron helper is nil",
			"response", response)
		return true
	}

	if time.Now().After(createNeuronHelper.deadline) {
		logger.Warn("error routing interaction create neuron response",
			"error", "interaction create neuron timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "neuron", correlationId)

	createNeuronHelper.interactionCreateChannel <- &neuronCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateAmdError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateAmdMapLock.Lock()
	createAmdHelper, exists := session.interactionCreateAmdMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateAmdMap, correlationId)
	}
	session.interactionCreateAmdMapLock.Unlock()

	if !exists {
		return false // not an AMD create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create AMD error response",
			"response", response)
	}

	// catch nil helper
	if createAmdHelper == nil {
		logger.Warn("error routing interaction create AMD response",
			"error", "AMD helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createAmdHelper.deadline) {
		logger.Warn("error routing interaction create AMD response",
			"error", "interaction create AMD timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "AMD", correlationId)

	// Send the error result through the channel
	createAmdHelper.interactionCreateChannel <- &amdCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateCpaError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateCpaMapLock.Lock()
	createCpaHelper, exists := session.interactionCreateCpaMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateCpaMap, correlationId)
	}
	session.interactionCreateCpaMapLock.Unlock()

	if !exists {
		return false // not a CPA create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create CPA error response",
			"response", response)
	}

	// catch nil helper
	if createCpaHelper == nil {
		logger.Warn("error routing interaction create CPA response",
			"error", "CPA helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createCpaHelper.deadline) {
		logger.Warn("error routing interaction create CPA response",
			"error", "interaction create CPA timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "CPA", correlationId)

	// Send the error result through the channel
	createCpaHelper.interactionCreateChannel <- &cpaCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateDiarizationError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateDiarizationMapLock.Lock()
	createDiarizationHelper, exists := session.interactionCreateDiarizationMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateDiarizationMap, correlationId)
	}
	session.interactionCreateDiarizationMapLock.Unlock()

	if !exists {
		return false // not a diarization create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create diarization error response",
			"response", response)
	}

	// catch nil helper
	if createDiarizationHelper == nil {
		logger.Warn("error routing interaction create diarization response",
			"error", "diarization helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createDiarizationHelper.deadline) {
		logger.Warn("error routing interaction create diarization response",
			"error", "interaction create diarization timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "diarization", correlationId)

	// Send the error result through the channel
	createDiarizationHelper.interactionCreateChannel <- &diarizationCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateGrammarParseError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateGrammarParseMapLock.Lock()
	createGrammarParseHelper, exists := session.interactionCreateGrammarParseMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateGrammarParseMap, correlationId)
	}
	session.interactionCreateGrammarParseMapLock.Unlock()

	if !exists {
		return false // not a grammar parse create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create grammar parse error response",
			"response", response)
	}

	// catch nil helper
	if createGrammarParseHelper == nil {
		logger.Warn("error routing interaction create grammar parse response",
			"error", "grammar parse helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createGrammarParseHelper.deadline) {
		logger.Warn("error routing interaction create grammar parse response",
			"error", "interaction create grammar parse timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "grammar parse", correlationId)

	// Send the error result through the channel
	createGrammarParseHelper.interactionCreateChannel <- &grammarParseCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateLanguageIdError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateLanguageIdMapLock.Lock()
	createLanguageIdHelper, exists := session.interactionCreateLanguageIdMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateLanguageIdMap, correlationId)
	}
	session.interactionCreateLanguageIdMapLock.Unlock()

	if !exists {
		return false // not a language id create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create language id error response",
			"response", response)
	}

	// catch nil helper
	if createLanguageIdHelper == nil {
		logger.Warn("error routing interaction create language id response",
			"error", "language id helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createLanguageIdHelper.deadline) {
		logger.Warn("error routing interaction create language id response",
			"error", "interaction create language id timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "language id", correlationId)

	// Send the error result through the channel
	createLanguageIdHelper.interactionCreateChannel <- &languageIdCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateNluError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateNluMapLock.Lock()
	createNluHelper, exists := session.interactionCreateNluMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateNluMap, correlationId)
	}
	session.interactionCreateNluMapLock.Unlock()

	if !exists {
		return false // not an NLU create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create NLU error response",
			"response", response)
	}

	// catch nil helper
	if createNluHelper == nil {
		logger.Warn("error routing interaction create NLU response",
			"error", "NLU helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createNluHelper.deadline) {
		logger.Warn("error routing interaction create NLU response",
			"error", "interaction create NLU timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "NLU", correlationId)

	// Send the error result through the channel
	createNluHelper.interactionCreateChannel <- &nluCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateNormalizationError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateNormalizeTextMapLock.Lock()
	createNormalizationHelper, exists := session.interactionCreateNormalizeTextMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateNormalizeTextMap, correlationId)
	}
	session.interactionCreateNormalizeTextMapLock.Unlock()

	if !exists {
		return false // not a normalization create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create normalization error response",
			"response", response)
	}

	// catch nil helper
	if createNormalizationHelper == nil {
		logger.Warn("error routing interaction create normalization response",
			"error", "normalization helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createNormalizationHelper.deadline) {
		logger.Warn("error routing interaction create normalization response",
			"error", "interaction create normalization timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "normalization", correlationId)

	// Send the error result through the channel
	createNormalizationHelper.interactionCreateChannel <- &normalizationCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateSentimentError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateSentimentMapLock.Lock()
	createSentimentHelper, exists := session.interactionCreateSentimentMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateSentimentMap, correlationId)
	}
	session.interactionCreateSentimentMapLock.Unlock()

	if !exists {
		return false // not a sentiment create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create sentiment error response",
			"response", response)
	}

	// catch nil helper
	if createSentimentHelper == nil {
		logger.Warn("error routing interaction create sentiment response",
			"error", "sentiment helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createSentimentHelper.deadline) {
		logger.Warn("error routing interaction create sentiment response",
			"error", "interaction create sentiment timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "sentiment", correlationId)

	// Send the error result through the channel
	createSentimentHelper.interactionCreateChannel <- &sentimentCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleInteractionCreateTtsError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionCreateTtsMapLock.Lock()
	createTtsHelper, exists := session.interactionCreateTtsMap[correlationId]
	if exists {
		// each helper should only be used once, so delete the map entry before unlocking
		delete(session.interactionCreateTtsMap, correlationId)
	}
	session.interactionCreateTtsMapLock.Unlock()

	if !exists {
		return false // not a TTS create error
	}

	if EnableVerboseLogging {
		logger.Debug("Recv interaction create TTS error response",
			"response", response)
	}

	// catch nil helper
	if createTtsHelper == nil {
		logger.Warn("error routing interaction create TTS response",
			"error", "TTS helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(createTtsHelper.deadline) {
		logger.Warn("error routing interaction create TTS response",
			"error", "interaction create TTS timed out",
			"response", response)
		return true
	}

	// Create error from API response
	apiError := interactionCreateError(response, "TTS", correlationId)

	// Send the error result through the channel
	createTtsHelper.interactionCreateChannel <- &ttsCreateResult{
		InteractionObject: nil,
		Error:             apiError,
	}
	return true
}

func handleFinalizeInteractionResponse(session *SessionObject, response *api.SessionResponse, correlationId string) bool {
	logger := getLogger()

	session.interactionFinalizeMapLock.Lock()
	finalizeHelper, exists := session.interactionFinalizeMap[correlationId]
	// each helper should only be used once, so delete the map entry before unlocking
	delete(session.interactionFinalizeMap, correlationId)
	session.interactionFinalizeMapLock.Unlock()

	if !exists {
		return false // not a finalize response
	}

	if EnableVerboseLogging {
		logger.Debug("Recv finalize interaction response",
			"response", response)
	}

	// catch nil helper
	if finalizeHelper == nil {
		logger.Warn("error routing finalize interaction response",
			"error", "finalize helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(finalizeHelper.deadline) {
		logger.Warn("error routing finalize interaction response",
			"error", "finalize interaction timed out",
			"response", response)
		return true
	}

	// determine success or failure based on status code
	var responseErr error
	if response.GetSessionEvent().StatusMessage != nil && response.GetSessionEvent().StatusMessage.Code != 0 {
		responseErr = fmt.Errorf("finalize interaction failed: %s (code: %d)",
			response.GetSessionEvent().StatusMessage.Message,
			response.GetSessionEvent().StatusMessage.Code)
	}

	// send the response (nil for success, error for failure)
	finalizeHelper.responseChannel <- responseErr
	return true
}

func handleSessionSetSettingsResponse(session *SessionObject, response *api.SessionResponse, correlationId string) bool {

	logger := getLogger()

	// check the coordination map for the correlationId
	session.sessionSetSettingsMapLock.Lock()
	setSettingsHelper, exists := session.sessionSetSettingsMap[correlationId]
	delete(session.sessionSetSettingsMap, correlationId)
	session.sessionSetSettingsMapLock.Unlock()

	if !exists {
		return false // not a set settings response
	}

	if EnableVerboseLogging {
		logger.Debug("Recv session set settings response",
			"response", response)
	}

	// catch nil helper
	if setSettingsHelper == nil {
		logger.Warn("error routing session set settings response",
			"error", "set settings helper is nil",
			"response", response)
		return true
	}

	// catch timeout
	if time.Now().After(setSettingsHelper.deadline) {
		logger.Warn("error routing session set settings response",
			"error", "session set settings timed out",
			"response", response)
		return true
	}

	// determine success or failure based on status code
	var responseErr error
	if response.GetSessionEvent().StatusMessage != nil && response.GetSessionEvent().StatusMessage.Code != 0 {
		responseErr = fmt.Errorf("session set settings failed: %s (code: %d)",
			response.GetSessionEvent().StatusMessage.Message,
			response.GetSessionEvent().StatusMessage.Code)
	}

	// send the response (nil for success, error for failure)
	setSettingsHelper.responseChannel <- responseErr
	return true
}

func handleAudioPullError(session *SessionObject, response *api.SessionResponse, correlationId string) bool {

	logger := getLogger()

	session.ttsAudioMapLock.Lock()
	audioPullChannel, exists := session.ttsAudioMap[correlationId]
	if !exists {
		session.ttsAudioMapLock.Unlock()
		return false // not an audio pull response
	}

	// the API may send empty SessionEvents with a zero status code. these are
	// not errors, so ignore them and leave the audio pull channel in place.
	statusMessage := response.GetSessionEvent().StatusMessage
	if statusMessage == nil || statusMessage.Code == 0 {
		session.ttsAudioMapLock.Unlock()
		if EnableVerboseLogging {
			logger.Debug("Recv audio pull session event with no error, ignoring",
				"response", response)
		}
		return true
	}

	// the audio pull has failed, so the channel won't be used again. delete
	// the map entry before unlocking.
	delete(session.ttsAudioMap, correlationId)
	session.ttsAudioMapLock.Unlock()

	logger.Error("audio pull failed",
		"error", statusMessage.Message,
		"code", statusMessage.Code,
		"correlation_id", correlationId)

	result := &audioPullResult{
		Response: nil,
		Error: fmt.Errorf("audio pull failed: %s (code: %d)",
			statusMessage.Message,
			statusMessage.Code),
	}

	// pass the error through the channel
	select {
	case audioPullChannel <- result:
	default:
		logger.Error("Recv audio pull error: audio pull channel full",
			"correlation_id", correlationId)
	}
	return true
}

func handleOtherSessionEvents(session *SessionObject, response *api.SessionResponse) {
	// Handle grammar errors and other non-correlation-based session events
	if response.GetSessionEvent().StatusMessage != nil {
		if strings.Contains(response.GetSessionEvent().StatusMessage.Message, "grammar failed to load") {
			logger := getLogger()
			logger.Error("Recv grammar error",
				"response", response)
			session.grammarErrorChannel <- response.GetSessionEvent()
		}
	}
}

func handleAudioPull(session *SessionObject, response *api.SessionResponse) {

	logger := getLogger()

	// get audio pull response
	audioPullResponse := response.GetAudioPull()
	if audioPullResponse == nil {
		logger.Error("Recv AudioPull: invalid response",
			"response", response)
		return
	}

	// attempt to get the correlationId
	if response.CorrelationId == nil || response.CorrelationId.Value == "" {
		logger.Error("Recv AudioPull: empty or missing correlation id",
			"response", response)
		return
	}
	correlationId := response.CorrelationId.Value

	// attempt to get the relevant audio pull channel
	session.ttsAudioMapLock.Lock()
	audioPullChannel, ok := session.ttsAudioMap[correlationId]
	if ok == false {
		logger.Error("Recv AudioPull: audio pull channel not found",
			"correlation_id", correlationId)
		session.ttsAudioMapLock.Unlock()
		return
	}

	// if this is a final audio chunk, go ahead and delete the channel from
	// the map before releasing the lock, because we won't be using it again
	if audioPullResponse.FinalDataChunk {
		delete(session.ttsAudioMap, correlationId)
	}
	session.ttsAudioMapLock.Unlock()

	// pass the audio response through the channel
	select {
	case audioPullChannel <- &audioPullResult{Response: audioPullResponse}:
	default:
		logger.Error("Recv AudioPull: audio pull channel full",
			"correlation_id", correlationId)
	}
}
