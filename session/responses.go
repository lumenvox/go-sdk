package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

// sessionResponseListener handles all responses from the specified session.
func sessionResponseListener(session *SessionObject, sessionIdChan chan string) {

	if EnableVerboseLogging {
		defer func() {
			log.Printf("exiting responseHandler for session: %s", session.SessionId)
		}()
	}

	waitingOnSessionId := true

	for {
		response, err := session.SessionStream.Recv()
		if err == io.EOF {
			if waitingOnSessionId {
				sessionIdChan <- ""
			}
			break
		}
		if err != nil {
			fmt.Printf("%s: sessionStream.Recv() failed: %v", time.Now().String(), err)
			if waitingOnSessionId {
				sessionIdChan <- ""
			}
			break
		}

		if EnableVerboseLogging {
			log.Printf("####>>>> sessionStream.Recv() received: %v", response)
		}

		if response.SessionId != nil && response.SessionId.Value != "" {
			if response.SessionId.Value != "" {
				if EnableVerboseLogging {
					log.Printf("Recv SessionId: %s", response.SessionId.Value)
				}
				if waitingOnSessionId {
					sessionIdChan <- response.SessionId.Value
					waitingOnSessionId = false
				} else {
					// if we weren't waiting on a session ID but got one anyway,
					// log a warning.
					log.Printf("warning: received extra session ID")
				}
			} else {
				log.Printf("Recv empty SessionId")
				if waitingOnSessionId {
					sessionIdChan <- ""
				}
				break
			}

		} else if response.GetVadEvent() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv VadEvent: %+v", response.GetVadEvent())
			}

			handleVadEvent(session, response)

		} else if response.GetPartialResult() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv PartialResult")
			}

			handlePartialResult(session, response)

		} else if response.GetFinalResult() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv FinalResult:\n%+v", response)
			}

			handleFinalResult(session, response)

		} else if response.GetInteractionCreateAsr() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv InteractionCreateAsrResponse: %+v", response.GetInteractionCreateAsr())
			}

			session.createdAsrChannel <- response.GetInteractionCreateAsr()

		} else if response.GetInteractionCreateTranscription() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv GetInteractionCreateTranscription: %+v", response.GetInteractionCreateTranscription())
			}

			session.createdTranscriptionChannel <- response.GetInteractionCreateTranscription()

		} else if response.GetInteractionCreateNlu() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv GetInteractionCreateNlu: %+v", response.GetInteractionCreateNlu())
			}

			session.createdNluChannel <- response.GetInteractionCreateNlu()

		} else if response.GetInteractionCreateNormalizeText() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv GetInteractionCreateNormalizeText: %+v", response.GetInteractionCreateNormalizeText())
			}

			session.createdNormalizeChannel <- response.GetInteractionCreateNormalizeText()

		} else if response.GetInteractionCreateGrammarParse() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv InteractionCreateGrammarParseResponse: %+v", response)
			}

			session.createdGrammarParseChannel <- response.GetInteractionCreateGrammarParse()

		} else if response.GetInteractionCreateTts() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv GetInteractionCreateTts: %+v", response)
			}

			session.createdTtsChannel <- response.GetInteractionCreateTts()

		} else if response.GetAudioPull() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv GetAudioPull numBytes: %d", len(response.GetAudioPull().GetAudioData()))
			}

			session.audioPullChannel <- response.GetAudioPull()

		} else if response.GetSessionGrammar() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv SessionLoadGrammarResponse: %+v", response)
			}

			session.sessionLoadChannel <- response.GetSessionGrammar()

		} else if response.GetSessionGetSettings() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv SessionGetSettings: %+v", response)
			}

			session.sessionSettingsChannel <- response.GetSessionGetSettings()

		} else {

			if response.GetSessionClose() != nil {

				if EnableVerboseLogging {
					log.Printf("Recv SessionCloseResponse: %+v", response)
				}
				session.SessionCloseChannel <- struct{}{}
				break

			} else if response.GetSessionEvent() != nil {

				if EnableVerboseLogging {
					log.Printf("Recv session event: %+v", response.GetSessionEvent())
				}
				if response.GetSessionEvent().StatusMessage != nil {
					if strings.Contains(response.GetSessionEvent().StatusMessage.Message, "grammar failed to load") {
						log.Printf("Recv grammar error: %+v", response)
						session.grammarErrorChannel <- response.GetSessionEvent()
					}
				}

			} else {
				log.Printf("Recv unexpected response type: %+v", response)
			}
		}
	}
}

func handleVadEvent(session *SessionObject, response *api.SessionResponse) {

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
				log.Printf("warning: VAD BARGE-IN event detected with empty AudioOffset. Using 0.")
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
				log.Printf("warning: received VAD event before BEGIN_PROCESSING: %v", response.GetVadEvent().VadEventType.String())
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
					log.Printf("warning: VAD BARGE-IN event detected with empty AudioOffset. Using 1.")
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
				log.Printf("warning: ignoring invalid VAD state transition: %v -> %v",
					currentVadInteractionState.String(),
					response.GetVadEvent().VadEventType.String())
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
					log.Printf("warning: VAD END_OF_SPEECH event detected with empty AudioOffset. Using 1.")
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
				log.Printf("warning: ignoring invalid VAD state transition: %v -> %v",
					currentVadInteractionState.String(),
					response.GetVadEvent().VadEventType.String())
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
				log.Printf("warning: ignoring invalid VAD state transition: %v -> %v",
					currentVadInteractionState.String(),
					response.GetVadEvent().VadEventType.String())
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
				log.Printf("warning: ignoring invalid VAD state transition: %v -> %v",
					currentVadInteractionState.String(),
					response.GetVadEvent().VadEventType.String())
			}

		default:
			// Unexpected current state. Should never happen, but just in case, log a warning
			log.Printf("warning: unexpected VAD current state: %v",
				currentVadInteractionState.String())
		}
		interactionObject.vadEventsLock.Unlock()

	} else {

		// We did not find the interaction.
		log.Printf("Recv VadEvent: interaction not found: %s", interactionId)

	}
}

func handleFinalResult(session *SessionObject, response *api.SessionResponse) {

	// Get the interaction id, to find the interaction object.
	interactionId := response.GetFinalResult().GetInteractionId()

	if interactionObject, ok := session.asrInteractionsMap[interactionId]; ok {

		// This is an ASR interaction.
		interactionObject.finalResults = response.GetFinalResult().GetFinalResult().GetAsrInteractionResult()
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)

	} else if interactionObject, ok := session.transcriptionInteractionsMap[interactionId]; ok {

		// This is a transcription interaction.
		interactionObject.finalResults = response.GetFinalResult().GetFinalResult().GetTranscriptionInteractionResult()
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)

	} else if interactionObject, ok := session.nluInteractionsMap[interactionId]; ok {

		// This is an NLU interaction.
		interactionObject.finalResults = response.GetFinalResult().GetFinalResult().GetNluInteractionResult()
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)

	} else if interactionObject, ok := session.normalizationInteractionsMap[interactionId]; ok {

		// This is a normalization interaction.
		interactionObject.finalResults = response.GetFinalResult().GetFinalResult().GetNormalizeTextResult()
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)

	} else if interactionObject, ok := session.ttsInteractionsMap[interactionId]; ok {

		// This is a TTS interaction.
		interactionObject.finalResults = response.GetFinalResult().GetFinalResult().GetTtsInteractionResult()
		interactionObject.finalResultsReceived = true
		close(interactionObject.resultsReadyChannel)

	} else {
		// We did not find the interaction.
		log.Printf("Recv FinalResult: interaction not found: %s", interactionId)
	}
}

func handlePartialResult(session *SessionObject, response *api.SessionResponse) {

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
		log.Printf("Recv PartialResult: interaction not found: %s", interactionId)
	}
}
