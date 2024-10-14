package session

import (
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "fmt"
    "log"
    "sync"
)

// vadInteractionRecord records the data from a single VAD interaction.
// For a successful interaction, it will store the values for barge-in and
// barge-out. If a timeout was received instead, the struct values will
// indicate that as well.
type vadInteractionRecord struct {
    beginProcessingReceived bool
    bargeInReceived         int
    bargeOutReceived        int
    bargeInTimeoutReceived  bool
}

func createEmptyVadInteractionRecord() *vadInteractionRecord {
    return &vadInteractionRecord{
        beginProcessingReceived: false,
        bargeInReceived:         0,
        bargeOutReceived:        0,
        bargeInTimeoutReceived:  false,
    }
}

// AsrInteractionObject represents an ASR interaction.
type AsrInteractionObject struct {
    InteractionId string

    // VAD begin processing tracking
    vadBeginProcessingReceived bool
    vadBeginProcessingChannel  chan struct{}

    // VAD barge-in tracking
    vadBargeInReceived int
    vadBargeInChannel  chan int

    // VAD barge-out tracking
    vadBargeOutReceived int
    vadBargeOutChannel  chan int

    // VAD barge-in timeout tracking
    vadBargeInTimeoutReceived bool
    vadBargeInTimeoutChannel  chan struct{}

    // Partial result tracking
    partialResultLock      sync.Mutex
    partialResultsReceived int
    partialResultsChannels []chan struct{}
    partialResultsList     []*api.PartialResult

    // Final result tracking
    finalResultsReceived bool
    finalResults         *api.AsrInteractionResult
    resultsReadyChannel  chan struct{}
}

// TranscriptionInteractionObject represents a transcription interaction.
type TranscriptionInteractionObject struct {
    InteractionId string

    // VAD event tracking
    vadEventsLock              sync.Mutex
    vadBeginProcessingChannels []chan struct{}
    vadBargeInChannels         []chan struct{}
    vadBargeOutChannels        []chan struct{}
    vadBargeInTimeoutChannels  []chan struct{}
    vadRecordLog               []*vadInteractionRecord
    vadInteractionCounter      int
    vadCurrentState            api.VadEvent_VadEventType

    // Partial result tracking
    partialResultLock      sync.Mutex
    partialResultsReceived int
    partialResultsChannels []chan struct{}
    partialResultsList     []*api.PartialResult

    // Final result tracking
    finalResultsReceived bool
    finalResults         *api.TranscriptionInteractionResult
    resultsReadyChannel  chan struct{}

    // For special result handling
    isContinuousTranscription bool
}

// NormalizationInteractionObject represents a normalization interaction.
type NormalizationInteractionObject struct {
    InteractionId string

    // Final result tracking
    finalResultsReceived bool
    finalResults         *api.NormalizeTextResult
    resultsReadyChannel  chan struct{}
}

// TtsInteractionObject represents a TTS interaction.
type TtsInteractionObject struct {
    InteractionId string

    // Final result tracking
    finalResultsReceived bool
    finalResults         *api.TtsInteractionResult
    resultsReadyChannel  chan struct{}
}

// NewAsr attempts to create a new ASR interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewAsr(
    language string,
    grammars []*api.Grammar,
    grammarSettings *api.GrammarSettings,
    recognitionSettings *api.RecognitionSettings,
    vadSettings *api.VadSettings,
    audioConsumeSettings *api.AudioConsumeSettings,
    generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *AsrInteractionObject, err error) {

    // Create ASR interaction, adding parameters such as VAD and recognition settings

    session.streamSendLock.Lock()
    err = session.SessionStream.Send(getAsrRequest("", language, grammars, grammarSettings,
        recognitionSettings, vadSettings, audioConsumeSettings, generalInteractionSettings))
    session.streamSendLock.Unlock()
    if err != nil {
        session.errorChan <- fmt.Errorf("sending InteractionCreateAsrRequest error: %v", err)
        log.Printf("error sending ASR create request: %v", err.Error())
        return nil, err
    }

    // Get the interaction ID.
    asrResponse := <-session.createdAsrChannel
    interactionId := asrResponse.InteractionId
    if EnableVerboseLogging {
        log.Printf("created new ASR interaction: %s", interactionId)
    }

    // Create the interaction object.
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

    // Add the interaction object to the session
    session.asrInteractionsMap[interactionId] = interactionObject

    return interactionObject, err
}

// NewTranscription attempts to create a new transcription interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewTranscription(
    language string,
    audioConsumeSettings *api.AudioConsumeSettings,
    normalizationSettings *api.NormalizationSettings,
    vadSettings *api.VadSettings,
    recognitionSettings *api.RecognitionSettings,
    languageModelName string,
    acousticModelName string,
    enablePostProcessing string,
    enableContinuousTranscription *api.OptionalBool) (interactionObject *TranscriptionInteractionObject, err error) {

    // Create transcription interaction, adding parameters such as VAD and recognition settings

    session.streamSendLock.Lock()
    err = session.SessionStream.Send(getTranscriptionRequest("", language,
        vadSettings, audioConsumeSettings, normalizationSettings, recognitionSettings,
        languageModelName, acousticModelName, enablePostProcessing, enableContinuousTranscription))
    session.streamSendLock.Unlock()
    if err != nil {
        session.errorChan <- fmt.Errorf("sending InteractionCreateTranscriptionRequest error: %v", err)
        log.Printf("error sending transcription create request: %v", err.Error())
        return nil, err
    }

    // Get the interaction ID.
    transcriptionResponse := <-session.createdTranscriptionChannel
    interactionId := transcriptionResponse.InteractionId
    if EnableVerboseLogging {
        log.Printf("created new transcription interaction: %s", interactionId)
    }

    // determine if this is a continuous interaction
    isContinuousTranscription := false
    if enableContinuousTranscription != nil && enableContinuousTranscription.Value {
        isContinuousTranscription = true
    }

    // Create the interaction object.
    interactionObject = &TranscriptionInteractionObject{
        InteractionId:        interactionId,
        finalResultsReceived: false,
        finalResults:         nil,
        resultsReadyChannel:  make(chan struct{}),
        vadCurrentState:      api.VadEvent_VAD_EVENT_TYPE_UNSPECIFIED,

        isContinuousTranscription: isContinuousTranscription,
    }
    interactionObject.partialResultsChannels = append(interactionObject.partialResultsChannels, make(chan struct{}))
    interactionObject.vadBeginProcessingChannels = append(interactionObject.vadBeginProcessingChannels, make(chan struct{}))
    interactionObject.vadBargeInChannels = append(interactionObject.vadBargeInChannels, make(chan struct{}))
    interactionObject.vadBargeOutChannels = append(interactionObject.vadBargeOutChannels, make(chan struct{}))
    interactionObject.vadBargeInTimeoutChannels = append(interactionObject.vadBargeInTimeoutChannels, make(chan struct{}))
    interactionObject.vadRecordLog = append(interactionObject.vadRecordLog, createEmptyVadInteractionRecord())

    // Add the interaction object to the session
    session.transcriptionInteractionsMap[interactionId] = interactionObject

    return interactionObject, err
}

// NewNormalization attempts to create a new ITN normalization interaction.
// If successful, a new interaction object will be returned.
func (session *SessionObject) NewNormalization(language string,
    textToNormalize string,
    normalizationSettings *api.NormalizationSettings,
    generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *NormalizationInteractionObject, err error) {

    // Create normalization interaction, adding specified parameters

    session.streamSendLock.Lock()
    err = session.SessionStream.Send(getNormalizationRequest("", language, textToNormalize,
        normalizationSettings, generalInteractionSettings))
    session.streamSendLock.Unlock()
    if err != nil {
        session.errorChan <- fmt.Errorf("sending InteractionCreateNormalizationRequest error: %v", err)
        log.Printf("error sending normalization create request: %v", err.Error())
        return nil, err
    }

    // Get the interaction ID.
    normalizationResponse := <-session.createdNormalizeChannel
    interactionId := normalizationResponse.InteractionId
    if EnableVerboseLogging {
        log.Printf("created new ITN interaction: %s", interactionId)
    }

    // Create the interaction object.
    interactionObject = &NormalizationInteractionObject{
        interactionId,
        false,
        nil,
        make(chan struct{}),
    }

    // Add the interaction object to the session
    session.normalizationInteractionsMap[interactionId] = interactionObject

    return interactionObject, err
}

// NewInlineTts attempts to create a new inline TTS interaction. If successful,
// a new interaction object will be returned.
func (session *SessionObject) NewInlineTts(language string,
    textToSynthesize string,
    inlineSettings *api.TtsInlineSynthesisSettings,
    synthesizedAudioFormat *api.AudioFormat,
    synthesisTimeoutMs *api.OptionalInt32,
    generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *TtsInteractionObject, err error) {

    // Create TTS interaction, adding specified parameters

    session.streamSendLock.Lock()
    err = session.SessionStream.Send(getInlineTtsRequest("", language, textToSynthesize,
        synthesizedAudioFormat, synthesisTimeoutMs, inlineSettings, generalInteractionSettings))
    session.streamSendLock.Unlock()
    if err != nil {
        session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
        log.Printf("error sending tts create request: %v", err.Error())
        return nil, err
    }

    // Get the interaction ID.
    ttsResponse := <-session.createdTtsChannel
    interactionId := ttsResponse.InteractionId
    if EnableVerboseLogging {
        log.Printf("created new TTS interaction: %s", interactionId)
    }

    // Create the interaction object.
    interactionObject = &TtsInteractionObject{
        interactionId,
        false,
        nil,
        make(chan struct{}),
    }

    // Add the interaction object to the session
    session.ttsInteractionsMap[interactionId] = interactionObject

    return interactionObject, err
}

// NewSsmlTts attempts to create a new SSML TTS interaction. If successful, a new interaction
// object will be returned.
func (session *SessionObject) NewSsmlTts(language string,
    textToSynthesize string,
    sslVerifyPeer *api.OptionalBool,
    synthesizedAudioFormat *api.AudioFormat,
    synthesisTimeoutMs *api.OptionalInt32,
    generalInteractionSettings *api.GeneralInteractionSettings) (interactionObject *TtsInteractionObject, err error) {

    // Create TTS interaction, adding specified parameters

    session.streamSendLock.Lock()
    err = session.SessionStream.Send(getSsmlTtsRequest("", language, textToSynthesize,
        synthesizedAudioFormat, synthesisTimeoutMs, sslVerifyPeer, generalInteractionSettings))
    session.streamSendLock.Unlock()
    if err != nil {
        session.errorChan <- fmt.Errorf("sending InteractionCreateTtsRequest error: %v", err)
        log.Printf("error sending tts create request: %v", err.Error())
        return nil, err
    }

    // Get the interaction ID.
    ttsResponse := <-session.createdTtsChannel
    interactionId := ttsResponse.InteractionId
    if EnableVerboseLogging {
        log.Printf("created new TTS interaction: %s", interactionId)
    }

    // Create the interaction object.
    interactionObject = &TtsInteractionObject{
        interactionId,
        false,
        nil,
        make(chan struct{}),
    }

    // Add the interaction object to the session
    session.ttsInteractionsMap[interactionId] = interactionObject

    return interactionObject, err
}

// PullTtsAudio fetches generated audio from a specified TTS interaction.
func (session *SessionObject) PullTtsAudio(interactionId string, audioChannel int32, audioStartMs int32,
    audioLengthMs int32) (audioData []byte, err error) {

    // Send audio pull request using the provided interactionId, adding specified parameters

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

// FinalizeInteraction attempts to finalize an existing interaction. This is not limited to a
// single interaction type.
func (session *SessionObject) FinalizeInteraction(interactionId string) (err error) {

    // Send finalize request using the provided interactionId, adding specified parameters

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
