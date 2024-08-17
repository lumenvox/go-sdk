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

	defer func() {
		if EnableVerboseLogging {
			log.Printf("exiting responseHandler for session: %s", session.SessionId)
		}
	}()

	for {
		response, err := session.SessionStream.Recv()
		if err == io.EOF {
			sessionIdChan <- ""
			break
		}
		if err != nil {
			fmt.Printf("%s: sessionStream.Recv() failed: %v", time.Now().String(), err)
			sessionIdChan <- ""
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
				sessionIdChan <- response.SessionId.Value
			} else {
				log.Printf("Recv empty SessionId")
				sessionIdChan <- ""
				break
			}

		} else if response.GetVadEvent() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv VadEvent: %+v", response.GetVadEvent())
			}

			handleVadEvent(session, response)

			session.VadMessagesChannel <- response.GetVadEvent()

		} else if response.GetPartialResult() != nil {
			if EnableVerboseLogging {
				log.Printf("Recv PartialResult")
			}

			handlePartialResult(session, response)

			session.PartialResultsChannel <- response.GetPartialResult()
		} else if response.GetFinalResult() != nil {

			if EnableVerboseLogging {
				log.Printf("Recv FinalResult:\n%+v", response)
			}

			handleFinalResult(session, response)

			session.FinalResultsChannel <- response.GetFinalResult()

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
		log.Printf("Recv FinalResult: interaction not found: %s", interactionId)
	}
}
