package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"fmt"
	"log"
	"sync"
)

type AsrInteractionObject struct {
	InteractionId string

	vadBeginProcessingReceived bool
	vadBeginProcessingChannel  chan struct{}

	vadBargeInReceived int
	vadBargeInChannel  chan int

	vadBargeOutReceived int
	vadBargeOutChannel  chan int

	vadBargeInTimeoutReceived bool
	vadBargeInTimeoutChannel  chan struct{}

	partialResultLock      sync.Mutex
	partialResultsReceived int
	partialResultsChannels []chan struct{}
	partialResultsList     []*api.PartialResult

	finalResultsReceived bool
	finalResults         *api.AsrInteractionResult
	resultsReadyChannel  chan struct{}
}

type TranscriptionInteractionObject struct {
	InteractionId string

	vadBeginProcessingReceived bool
	vadBeginProcessingChannel  chan struct{}

	vadBargeInReceived int
	vadBargeInChannel  chan int

	vadBargeOutReceived int
	vadBargeOutChannel  chan int

	vadBargeInTimeoutReceived bool
	vadBargeInTimeoutChannel  chan struct{}

	partialResultLock      sync.Mutex
	partialResultsReceived int
	partialResultsChannels []chan struct{}
	partialResultsList     []*api.PartialResult

	finalResultsReceived bool
	finalResults         *api.TranscriptionInteractionResult
	resultsReadyChannel  chan struct{}
}

type NormalizationInteractionObject struct {
	InteractionId string

	finalResultsReceived bool
	finalResults         *api.NormalizeTextResult
	resultsReadyChannel  chan struct{}
}

type TtsInteractionObject struct {
	InteractionId string

	finalResultsReceived bool
	finalResults         *api.TtsInteractionResult
	resultsReadyChannel  chan struct{}
}

func (session *SessionObject) NewAsr(
	language string,
	grammars []*api.Grammar,
	grammarSettings *api.GrammarSettings,
	recognitionSettings *api.RecognitionSettings,
	vadSettings *api.VadSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *AsrInteractionObject, err error) {

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getAsrRequest("", language, grammars, grammarSettings,
		recognitionSettings, vadSettings, audioConsumeSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateAsrRequest error: %v", err)
		log.Printf("error sending ASR create request: %v", err.Error())
		return nil, err
	}

	asrResponse := <-session.createdAsrChannel
	interactionId := asrResponse.InteractionId
	if EnableVerboseLogging {
		log.Printf("received asr response: %s", interactionId)
	}

	interactionObject = &AsrInteractionObject{
		InteractionId:             interactionId,
		vadBeginProcessingChannel: make(chan struct{}, 1),
		vadBargeInChannel:         make(chan int, 1),
		vadBargeInReceived:        -1,
		vadBargeOutChannel:        make(chan int, 1),
		vadBargeOutReceived:       -1,
		vadBargeInTimeoutChannel:  make(chan struct{}),
		vadBargeInTimeoutReceived: false,
		finalResultsReceived:      false,
		finalResults:              nil,
		resultsReadyChannel:       make(chan struct{}),
	}
	interactionObject.partialResultsChannels = append(interactionObject.partialResultsChannels, make(chan struct{}))
	session.asrInteractionsMap[interactionId] = interactionObject

	return interactionObject, err
}

func (session *SessionObject) NewTranscription(
	language string,
	audioConsumeSettings *api.AudioConsumeSettings,
	normalizationSettings *api.NormalizationSettings,
	vadSettings *api.VadSettings,
	recognitionSettings *api.RecognitionSettings) (interactionObject *TranscriptionInteractionObject, err error) {

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getTranscriptionRequest("", language,
		vadSettings, audioConsumeSettings, normalizationSettings, recognitionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateTranscriptionRequest error: %v", err)
		log.Printf("error sending transcription create request: %v", err.Error())
		return nil, err
	}

	transcriptionResponse := <-session.createdTranscriptionChannel
	interactionId := transcriptionResponse.InteractionId
	if EnableVerboseLogging {
		log.Printf("received transcription response: %s", interactionId)
	}

	interactionObject = &TranscriptionInteractionObject{
		InteractionId:             interactionId,
		vadBeginProcessingChannel: make(chan struct{}, 1),
		vadBargeInChannel:         make(chan int, 1),
		vadBargeInReceived:        -1,
		vadBargeOutChannel:        make(chan int, 1),
		vadBargeOutReceived:       -1,
		vadBargeInTimeoutChannel:  make(chan struct{}),
		vadBargeInTimeoutReceived: false,
		finalResultsReceived:      false,
		finalResults:              nil,
		resultsReadyChannel:       make(chan struct{}),
	}
	interactionObject.partialResultsChannels = append(interactionObject.partialResultsChannels, make(chan struct{}))
	session.transcriptionInteractionsMap[interactionId] = interactionObject

	return interactionObject, err
}

func (session *SessionObject) NewNormalization(language string,
	textToNormalize string,
	normalizationSettings *api.NormalizationSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *NormalizationInteractionObject, err error) {

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getNormalizationRequest("", language, textToNormalize,
		normalizationSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateNormalizationRequest error: %v", err)
		log.Printf("error sending normalization create request: %v", err.Error())
		return nil, err
	}

	normalizationResponse := <-session.createdNormalizeChannel
	interactionId := normalizationResponse.InteractionId
	log.Printf("received normalization response: %s", interactionId)

	interactionObject = &NormalizationInteractionObject{
		interactionId,
		false,
		nil,
		make(chan struct{}),
	}
	session.normalizationInteractionsMap[interactionId] = interactionObject

	return interactionObject, err
}

func (session *SessionObject) NewInlineTts(language string,
	textToSynthesize string,
	inlineSettings *api.TtsInlineSynthesisSettings,
	synthesizedAudioFormat *api.AudioFormat,
	synthesisTimeoutMs *api.OptionalInt32,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *TtsInteractionObject, err error) {

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getInlineTtsRequest("", language, textToSynthesize,
		synthesizedAudioFormat, synthesisTimeoutMs, inlineSettings, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
		log.Printf("error sending tts create request: %v", err.Error())
		return nil, err
	}

	ttsResponse := <-session.createdTtsChannel
	interactionId := ttsResponse.InteractionId
	log.Printf("received tts response: %s", interactionId)

	interactionObject = &TtsInteractionObject{
		interactionId,
		false,
		nil,
		make(chan struct{}),
	}
	session.ttsInteractionsMap[interactionId] = interactionObject

	return interactionObject, err
}

func (session *SessionObject) NewSsmlTts(language string,
	textToSynthesize string,
	sslVerifyPeer *api.OptionalBool,
	synthesizedAudioFormat *api.AudioFormat,
	synthesisTimeoutMs *api.OptionalInt32,
	generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *TtsInteractionObject, err error) {

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getSsmlTtsRequest("", language, textToSynthesize,
		synthesizedAudioFormat, synthesisTimeoutMs, sslVerifyPeer, generalInteractionSettings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
		log.Printf("error sending tts create request: %v", err.Error())
		return nil, err
	}

	ttsResponse := <-session.createdTtsChannel
	interactionId := ttsResponse.InteractionId
	log.Printf("received tts response: %s", interactionId)

	interactionObject = &TtsInteractionObject{
		interactionId,
		false,
		nil,
		make(chan struct{}),
	}
	session.ttsInteractionsMap[interactionId] = interactionObject

	return interactionObject, err
}

func (session *SessionObject) PullTtsAudio(interactionId string, audioChannel int32, audioStartMs int32,
	audioLengthMs int32) (audioData []byte, err error) {

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getAudioPullRequest("", interactionId, audioChannel, audioStartMs, audioLengthMs))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending AudioPullRequest error: %v", err)
		log.Printf("error sending audio pull request: %v", err.Error())
		return nil, err
	}

	audioDataResponse := <-session.audioPullChannel
	audioData = audioDataResponse.GetAudioData()

	return audioData, nil
}

func (session *SessionObject) FinalizeInteraction(interactionId string) (err error) {

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getInteractionFinalizeRequest("", interactionId))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionFinalizeProcessingRequest error: %v", err)
		log.Printf("error sending finalize processing request: %v", err.Error())
		return err
	}

	return nil
}
