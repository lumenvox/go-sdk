package session

import (
    "context"
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "errors"
    "fmt"
    "google.golang.org/grpc"
    "log"
    "sync"
    "time"
)

// EnableVerboseLogging generates debugging logs for developers.
var EnableVerboseLogging = false

// SessionObject contains all state information for a single session.
type SessionObject struct {
    sync.RWMutex

    // session/stream information
    streamContext context.Context
    SessionCancel context.CancelFunc
    SessionStream api.LumenVox_SessionClient
    audioConfig   AudioConfig
    SessionId     string
    StreamTimeout time.Duration
    DeploymentId  string
    OperatorId    string

    // streaming control
    streamSendLock sync.Mutex // protects against concurrent writes to stream

    // audio streaming status
    audioStreamerLock        sync.Mutex
    audioStreamerInitialized bool
    audioStreamerCancel      context.CancelFunc

    audioLock      sync.Mutex
    audioForStream []byte

    // channels
    VadMessagesChannel          chan *api.VadEvent
    PartialResultsChannel       chan *api.PartialResult
    FinalResultsChannel         chan *api.FinalResult
    audioPullChannel            chan *api.AudioPullResponse
    sessionLoadChannel          chan *api.SessionLoadGrammarResponse
    SessionCloseChannel         chan struct{}
    createdAsrChannel           chan *api.InteractionCreateAsrResponse
    createdTranscriptionChannel chan *api.InteractionCreateTranscriptionResponse
    createdNormalizeChannel     chan *api.InteractionCreateNormalizeTextResponse
    createdGrammarParseChannel  chan *api.InteractionCreateGrammarParseResponse
    createdTtsChannel           chan *api.InteractionCreateTtsResponse
    grammarErrorChannel         chan *api.SessionEvent
    errorChan                   chan error
    stopStreamingAudio          chan bool

    // maps to store all active interactions
    asrInteractionsMap           map[string]*AsrInteractionObject
    transcriptionInteractionsMap map[string]*TranscriptionInteractionObject
    normalizationInteractionsMap map[string]*NormalizationInteractionObject
    ttsInteractionsMap           map[string]*TtsInteractionObject

    // settings
    sessionSettingsChannel chan *api.SessionSettings
}

// SessionMapObject is used to keep track of all open sessions
type SessionMapObject struct {
    sync.RWMutex
    OpenSessions map[string]*SessionObject
}

// activeSessionsMap holds information about all current sessions
var activeSessionsMap = SessionMapObject{
    OpenSessions: make(map[string]*SessionObject),
}

func newSessionObject(
    streamContext context.Context,
    sessionStream api.LumenVox_SessionClient,
    streamCancel context.CancelFunc,
    streamTimeout time.Duration,
    deploymentId, operatorId string,
    audioConfig AudioConfig,
) *SessionObject {

    return &SessionObject{
        streamContext: streamContext,
        SessionStream: sessionStream,
        SessionCancel: streamCancel,
        StreamTimeout: streamTimeout,
        DeploymentId:  deploymentId,
        OperatorId:    operatorId,
        audioConfig:   audioConfig,

        // channels
        VadMessagesChannel:          make(chan *api.VadEvent, 100),
        PartialResultsChannel:       make(chan *api.PartialResult, 100),
        FinalResultsChannel:         make(chan *api.FinalResult, 100),
        audioPullChannel:            make(chan *api.AudioPullResponse, 100),
        sessionLoadChannel:          make(chan *api.SessionLoadGrammarResponse, 100),
        SessionCloseChannel:         make(chan struct{}, 1),
        createdAsrChannel:           make(chan *api.InteractionCreateAsrResponse, 100),
        createdTranscriptionChannel: make(chan *api.InteractionCreateTranscriptionResponse, 100),
        createdNormalizeChannel:     make(chan *api.InteractionCreateNormalizeTextResponse, 100),
        createdGrammarParseChannel:  make(chan *api.InteractionCreateGrammarParseResponse, 100),
        createdTtsChannel:           make(chan *api.InteractionCreateTtsResponse, 100),
        grammarErrorChannel:         make(chan *api.SessionEvent, 100),
        errorChan:                   make(chan error, 10),
        stopStreamingAudio:          make(chan bool, 1),
        sessionSettingsChannel:      make(chan *api.SessionSettings, 100),

        // interaction maps
        asrInteractionsMap:           make(map[string]*AsrInteractionObject),
        transcriptionInteractionsMap: make(map[string]*TranscriptionInteractionObject),
        normalizationInteractionsMap: make(map[string]*NormalizationInteractionObject),
        ttsInteractionsMap:           make(map[string]*TtsInteractionObject),
    }
}

// CreateNewSession creates a new session object that can be used to create and run interactions.
func CreateNewSession(
    clientConn *grpc.ClientConn,
    streamTimeout time.Duration,
    deploymentId string,
    audioConfig AudioConfig,
    operatorId string,
) (newSession *SessionObject, err error) {

    // Set up the stream.
    grpcClient := api.NewLumenVoxClient(clientConn)
    streamContext, streamCancel := context.WithTimeout(context.Background(), streamTimeout)

    // Create the sessionStream.
    sessionStream, err := grpcClient.Session(streamContext)

    // Catch any errors from the stream.
    if err != nil {
        streamCancel()
        return nil, err
    }

    // Set up the session config.
    newSession = newSessionObject(streamContext, sessionStream, streamCancel, streamTimeout,
        deploymentId, operatorId, audioConfig)

    // Set up a channel to get the session ID.
    sessionIdChan := make(chan string, 1)

    // Start the response listener to take responses from the stream and filter
    // them into corresponding channels. This runs in the background for the
    // duration of the session.
    go sessionResponseListener(newSession, sessionIdChan)

    // Request new session from API.
    newSession.streamSendLock.Lock()
    err = newSession.SessionStream.Send(getSessionCreateRequest("",
        newSession.DeploymentId, newSession.OperatorId))
    newSession.streamSendLock.Unlock()
    if err != nil {
        // If we failed to send the request, log an error, cancel the stream,
        // and return.
        log.Printf("stream.Send failed sending session request: %v\n", err)
        streamCancel()
        return nil, err
    }

    // Get session id from channel (should be populated by the response
    // listener).
    var newSessionId string
    select {
    case newSessionId = <-sessionIdChan:
    case <-time.After(10 * time.Second):
        // No session ID for 10 seconds, give up.
        log.Printf("no session ID for 10 seconds")
        streamCancel()
        return nil, errors.New("timeout waiting for session ID")
    }

    // Validate the session ID.
    if newSessionId != "" {
        // If we got an ID, update the session object before adding it to the sessions map.
        newSession.Lock()
        newSession.SessionId = newSessionId
        newSession.Unlock()

        activeSessionsMap.Lock()
        activeSessionsMap.OpenSessions[newSessionId] = newSession
        activeSessionsMap.Unlock()
    } else {
        // We got an empty session ID. clean up.
        log.Printf("empty session ID")
        streamCancel()
        return nil, errors.New("empty session ID")
    }

    // Set audio format if not NO_AUDIO_RESOURCE.
    if audioConfig.Format != api.AudioFormat_STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE {
        newSession.streamSendLock.Lock()
        err = newSession.SessionStream.Send(getAudioFormatRequest("", audioConfig))
        newSession.streamSendLock.Unlock()
        if err != nil {
            newSession.errorChan <- fmt.Errorf("sending SessionInboundAudioFormatRequest error: %v", err)
            log.Printf("stream.Send failed setting audio format: %v\n", err)
            newSession.SessionCancel()
            return nil, err
        }
    }

    return newSession, nil
}

// CloseSession closes the specified session object and releases its resources.
func (session *SessionObject) CloseSession() {

    sessionId := session.SessionId

    session.streamSendLock.Lock()
    err := session.SessionStream.Send(getSessionCloseRequest(""))
    session.streamSendLock.Unlock()
    if err != nil {
        log.Printf("error sending session close request (sessionId %s): %v", sessionId, err.Error())
    }

    // Pause to allow the close session message to be sent.
    time.Sleep(500 * time.Millisecond)

    session.SessionCancel()

    activeSessionsMap.Lock()
    delete(activeSessionsMap.OpenSessions, sessionId)
    activeSessionsMap.Unlock()

}
