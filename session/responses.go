package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"strings"
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
		response, err = session.SessionStream.Recv()
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
			logger.Debug("sessionStream.Recv() received",
				"response", response)
		}

		if response.SessionId != nil && response.SessionId.Value != "" {
			if response.SessionId.Value != "" {
				if EnableVerboseLogging {
					logger.Debug("Recv response",
						"SessionId", response.SessionId.Value)
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

		} else if response.GetVadEvent() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv VadEvent",
					"interactionId", response.GetVadEvent().InteractionId,
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

		} else if response.GetInteractionCreateAmd() != nil ||
			response.GetInteractionCreateAsr() != nil ||
			response.GetInteractionCreateCpa() != nil ||
			response.GetInteractionCreateDiarization() != nil ||
			response.GetInteractionCreateGrammarParse() != nil ||
			response.GetInteractionCreateLanguageId() != nil ||
			response.GetInteractionCreateNlu() != nil ||
			response.GetInteractionCreateNormalizeText() != nil ||
			response.GetInteractionCreateTranscription() != nil ||
			response.GetInteractionCreateTts() != nil {

			if EnableVerboseLogging {
				interactionType := "unknown"
				if response.GetInteractionCreateAmd() != nil {
					interactionType = "InteractionCreateAmdResponse"
				} else if response.GetInteractionCreateAsr() != nil {
					interactionType = "InteractionCreateAsrResponse"
				} else if response.GetInteractionCreateCpa() != nil {
					interactionType = "InteractionCreateCpaResponse"
				} else if response.GetInteractionCreateDiarization() != nil {
					interactionType = "InteractionCreateDiarizationResponse"
				} else if response.GetInteractionCreateGrammarParse() != nil {
					interactionType = "InteractionCreateGrammarParseResponse"
				} else if response.GetInteractionCreateLanguageId() != nil {
					interactionType = "InteractionCreateLanguageIdResponse"
				} else if response.GetInteractionCreateNlu() != nil {
					interactionType = "InteractionCreateNluResponse"
				} else if response.GetInteractionCreateNormalizeText() != nil {
					interactionType = "InteractionCreateNormalizeTextResponse"
				} else if response.GetInteractionCreateTranscription() != nil {
					interactionType = "InteractionCreateTranscriptionResponse"
				} else if response.GetInteractionCreateTts() != nil {
					interactionType = "InteractionCreateTtsResponse"
				}
				logger.Debug("recv interaction create response",
					"type", interactionType,
					"response", response)
			}

			// validate the correlation ID
			if response.CorrelationId == nil {
				logger.Warn("error routing interaction create",
					"error", "missing correlationId",
					"response", response)
				continue
			} else if response.CorrelationId.Value == "" {
				logger.Warn("error routing interaction create",
					"error", "empty correlationId",
					"response", response)
				continue
			}
			correlationId := response.CorrelationId.Value

			// attempt to send the response to the channel mapped by the correlationId
			err := session.signalInteractionCreate(correlationId, response)
			if err != nil {
				logger.Warn("error routing interaction create",
					"error", err.Error(),
					"response", response)
				continue
			}

		} else if response.GetAudioPull() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv GetAudioPull",
					"numBytes", len(response.GetAudioPull().GetAudioData()))
			}

			session.audioPullChannel <- response.GetAudioPull()

		} else if response.GetSessionGrammar() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv SessionLoadGrammarResponse",
					"response", response)
			}

			session.sessionLoadChannel <- response.GetSessionGrammar()

		} else if response.GetSessionGetSettings() != nil {

			if EnableVerboseLogging {
				logger.Debug("Recv SessionGetSettings",
					"response", response)
			}

			session.sessionSettingsChannel <- response.GetSessionGetSettings()

		} else {

			if response.GetSessionClose() != nil {

				if EnableVerboseLogging {
					logger.Debug("Recv SessionCloseResponse",
						"response", response)
				}
				session.SessionCloseChannel <- struct{}{}
				err = nil
				break

			} else if response.GetSessionEvent() != nil {

				if EnableVerboseLogging {
					logger.Debug("Recv session event",
						"response", response.GetSessionEvent())
				}
				if response.GetSessionEvent().StatusMessage != nil {
					if strings.Contains(response.GetSessionEvent().StatusMessage.Message, "grammar failed to load") {
						logger.Error("Recv grammar error",
							"response", response)
						session.grammarErrorChannel <- response.GetSessionEvent()
					}
				}

			} else {
				logger.Error("Recv unexpected response type",
					"response", response)
			}
		}
	}

	// Signal to session object here that we're out of the listening loop
	// and set the reason that loop exited (could be nil if no error)
	session.StreamLoopExitErr.Store(&err)
}

func handleVadEvent(session *SessionObject, response *api.SessionResponse) {

	logger := getLogger()

	// Get the interaction id, to find the interaction object.
	interactionId := response.GetVadEvent().GetInteractionId()

	if interactionObject, ok := session.asrInteractionsMap[interactionId]; ok {

		// This is an ASR interaction.
		switch response.GetVadEvent().VadEventType {
		case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
			interactionObject.vadBeginProcessingReceived = true
			interactionObject.vadBeginProcessingChannel <- struct{}{}
		case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN:
			bargeInAudioOffset := response.GetVadEvent().AudioOffset
			var bargeInTime int
			if bargeInAudioOffset == nil {
				bargeInTime = 0
				logger.Warn("VAD BARGE-IN event detected with empty AudioOffset. Using 0.")
			} else {
				bargeInTime = int(bargeInAudioOffset.Value)
			}
			interactionObject.vadBargeInReceived = bargeInTime
			interactionObject.vadBargeInChannel <- bargeInTime
		case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH:
			bargeOutTime := int(response.GetVadEvent().AudioOffset.Value)
			interactionObject.vadBargeOutReceived = bargeOutTime
			interactionObject.vadBargeOutChannel <- bargeOutTime
		case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN_TIMEOUT:
			interactionObject.vadBargeInTimeoutReceived = true
			close(interactionObject.vadBargeInTimeoutChannel)
		}

	} else if interactionObject, ok := session.transcriptionInteractionsMap[interactionId]; ok {

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
					"fromState", currentVadInteractionState.String(),
					"toState", response.GetVadEvent().VadEventType.String())
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

			// TODO: support more VAD cases here
			default:
				// We received something that doesn't make sense after a BARGE_IN. In other
				// words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"fromState", currentVadInteractionState.String(),
					"toState", response.GetVadEvent().VadEventType.String())
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
				interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBeginProcessingChannels[currentVadInteractionIndex])

			default:
				// We received something that doesn't make sense after an END_OF_SPEECH. In
				// other words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"fromState", currentVadInteractionState.String(),
					"toState", response.GetVadEvent().VadEventType.String())
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
				interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())
				// Now that the event has been recorded, signal the channel
				close(interactionObject.vadBeginProcessingChannels[currentVadInteractionIndex])

			default:
				// We received something that doesn't make sense after a BARGE_IN_TIMEOUT. In
				// other words, an invalid transition. Log a warning.
				logger.Warn("ignoring invalid VAD state transition",
					"fromState", currentVadInteractionState.String(),
					"toState", response.GetVadEvent().VadEventType.String())
			}

		default:
			// Unexpected current state. Should never happen, but just in case, log a warning
			logger.Warn("unexpected VAD",
				"current state", currentVadInteractionState.String())
		}
		interactionObject.vadEventsLock.Unlock()

	} else if interactionObject, ok := session.amdInteractionsMap[interactionId]; ok {
		// This is an AMD interaction.
		switch response.GetVadEvent().VadEventType {
		case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
			interactionObject.vadBeginProcessingReceived = true
			interactionObject.vadBeginProcessingChannel <- struct{}{}
		case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN:
			bargeInAudioOffset := response.GetVadEvent().AudioOffset
			var bargeInTime int
			if bargeInAudioOffset == nil {
				bargeInTime = 0
				logger.Warn("VAD BARGE-IN event detected with empty AudioOffset. Using 0.")
			} else {
				bargeInTime = int(bargeInAudioOffset.Value)
			}
			interactionObject.vadBargeInReceived = bargeInTime
			interactionObject.vadBargeInChannel <- bargeInTime
		case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH:
			bargeOutTime := int(response.GetVadEvent().AudioOffset.Value)
			interactionObject.vadBargeOutReceived = bargeOutTime
			interactionObject.vadBargeOutChannel <- bargeOutTime
		case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN_TIMEOUT:
			interactionObject.vadBargeInTimeoutReceived = true
			close(interactionObject.vadBargeInTimeoutChannel)
		}

	} else {

		// We did not find the interaction.
		logger.Error("Recv VadEvent: interaction not found",
			"interactionId", interactionId)

	}
}

func handleFinalResult(session *SessionObject, response *api.SessionResponse) {

	logger := getLogger()

	// Get the interaction id, to find the interaction object.
	interactionId := response.GetFinalResult().GetInteractionId()
	finalResult := response.GetFinalResult()

	{
		// Protect concurrent access to maps
		session.Lock()
		defer session.Unlock()

		if interactionObject, interactionFound := session.asrInteractionsMap[interactionId]; interactionFound {

			// This is an ASR interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetAsrInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.transcriptionInteractionsMap[interactionId]; interactionFound {

			// This is a transcription interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetTranscriptionInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.nluInteractionsMap[interactionId]; interactionFound {

			// This is an NLU interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetNluInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.amdInteractionsMap[interactionId]; interactionFound {

			// This is an AMD interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetAmdInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.cpaInteractionsMap[interactionId]; interactionFound {

			// This is a CPA interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetCpaInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.normalizationInteractionsMap[interactionId]; interactionFound {

			// This is a normalization interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetNormalizeTextResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.ttsInteractionsMap[interactionId]; interactionFound {

			// This is a TTS interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetTtsInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.diarizationInteractionsMap[interactionId]; interactionFound {

			// This is a diarization interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetDiarizationInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else if interactionObject, interactionFound := session.languageIdInteractionsMap[interactionId]; interactionFound {

			// This is a diarization interaction.
			interactionObject.finalResults = finalResult.GetFinalResult().GetLanguageIdInteractionResult()
			interactionObject.FinalStatus = finalResult.Status
			interactionObject.FinalResultStatus = finalResult.FinalResultStatus
			interactionObject.finalResultsReceived = true
			close(interactionObject.resultsReadyChannel)

		} else {
			// We did not find the interaction.
			logger.Error("Recv FinalResult: interaction not found",
				"interactionId", interactionId)
		}
	}
}

func handlePartialResult(session *SessionObject, response *api.SessionResponse) {

	logger := getLogger()

	// Get the interaction id, to find the interaction object.
	interactionId := response.GetPartialResult().GetInteractionId()

	if interactionObject, ok := session.asrInteractionsMap[interactionId]; ok {

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

	} else if interactionObject, ok := session.transcriptionInteractionsMap[interactionId]; ok {

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

	} else {
		// We did not find the interaction.
		logger.Error("Recv PartialResult: interaction not found",
			"interactionId", interactionId)
	}
}
