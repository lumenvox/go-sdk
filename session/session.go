package session

import (
	"github.com/lumenvox/go-sdk/auth"
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// TimeoutError represents a specific SDK error indicating that a timeout has occurred during an operation.
// This concerns timeouts while waiting for API responses, not recognition-based timeouts (like barge-in
// and barge-out timeouts).
const TimeoutError = SdkError("timeout waiting for API response")

type SdkError string

func (e SdkError) Error() string { return string(e) }

var EnableVerboseLogging = func() bool {
	return os.Getenv("LUMENVOX_GO_SDK__ENABLE_VERBOSE_LOGGING") == "true"
}()

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

	StreamLoopExitErr atomic.Pointer[error] // indicates streaming loop exit status

	// channels
	audioPullChannel    chan *api.AudioPullResponse
	sessionLoadChannel  chan *api.SessionLoadGrammarResponse
	SessionCloseChannel chan struct{}
	grammarErrorChannel chan *api.SessionEvent
	errorChan           chan error
	stopStreamingAudio  chan struct{}

	// map of channels to handle interactionCreate responses
	interactionCreateMapLock sync.Mutex
	interactionCreateMap     map[string]chan *api.SessionResponse

	// maps to store all active interactions
	asrInteractionsMap           map[string]*AsrInteractionObject
	transcriptionInteractionsMap map[string]*TranscriptionInteractionObject
	amdInteractionsMap           map[string]*AmdInteractionObject
	cpaInteractionsMap           map[string]*CpaInteractionObject
	nluInteractionsMap           map[string]*NluInteractionObject
	normalizationInteractionsMap map[string]*NormalizationInteractionObject
	ttsInteractionsMap           map[string]*TtsInteractionObject
	diarizationInteractionsMap   map[string]*DiarizationInteractionObject
	languageIdInteractionsMap    map[string]*LanguageIdInteractionObject

	// settings
	sessionSettingsChannel chan *api.SessionSettings
}

// sessionMapObject is used to keep track of all open sessions
type sessionMapObject struct {
	sync.RWMutex
	OpenSessions map[string]*SessionObject
}

// activeSessionsMap holds information about all current sessions
var activeSessionsMap = sessionMapObject{
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
		audioPullChannel:       make(chan *api.AudioPullResponse, 100),
		sessionLoadChannel:     make(chan *api.SessionLoadGrammarResponse, 100),
		SessionCloseChannel:    make(chan struct{}, 1),
		grammarErrorChannel:    make(chan *api.SessionEvent, 100),
		errorChan:              make(chan error, 10),
		stopStreamingAudio:     make(chan struct{}, 1),
		sessionSettingsChannel: make(chan *api.SessionSettings, 100),

		// map of channels to handle interactionCreate responses
		interactionCreateMap: make(map[string]chan *api.SessionResponse),

		// interaction maps
		asrInteractionsMap:           make(map[string]*AsrInteractionObject),
		transcriptionInteractionsMap: make(map[string]*TranscriptionInteractionObject),
		amdInteractionsMap:           make(map[string]*AmdInteractionObject),
		cpaInteractionsMap:           make(map[string]*CpaInteractionObject),
		nluInteractionsMap:           make(map[string]*NluInteractionObject),
		normalizationInteractionsMap: make(map[string]*NormalizationInteractionObject),
		ttsInteractionsMap:           make(map[string]*TtsInteractionObject),
		diarizationInteractionsMap:   make(map[string]*DiarizationInteractionObject),
		languageIdInteractionsMap:    make(map[string]*LanguageIdInteractionObject),
	}
}

// CreateNewSession creates a new session object which can be used to create and run interactions.
func CreateNewSession(
	clientConn *grpc.ClientConn,
	streamTimeout time.Duration,
	deploymentId string,
	audioConfig AudioConfig,
	operatorId string,
	authSettings *auth.AuthSettings,
) (newSession *SessionObject, err error) {

	logger := getLogger()

	var sessionStream api.LumenVox_SessionClient

	// only used when auth enabled...
	var authToken string
	var metaAuth metadata.MD

	// connectionContext will include auth if enabled
	var connectionContext context.Context

	// Set up the stream.
	grpcClient := api.NewLumenVoxClient(clientConn)
	streamContext, streamCancel := context.WithTimeout(context.Background(), streamTimeout)

	if authSettings != nil {
		// Authentication enabled

		tokenProvider := auth.GetGlobalCognitoProvider(*authSettings)

		authToken, err = tokenProvider.GetToken(streamContext)
		if err != nil {
			streamCancel()
			return nil, err
		}

		// Add an Authorization header to the request
		metaAuth = metadata.Pairs("Authorization", "Bearer "+authToken)

		connectionContext = metadata.NewOutgoingContext(streamContext, metaAuth)
	} else {
		connectionContext = streamContext
	}

	// Create the sessionStream.
	sessionStream, err = grpcClient.Session(connectionContext)

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
		logger.Error("stream.Send failed sending session request",
			"error", err)
		streamCancel()
		return nil, errors.New("failed to send session request")
	}

	// Get session id from channel (should be populated by the response
	// listener).
	var newSessionId string
	select {
	case newSessionId = <-sessionIdChan:
	case <-time.After(10 * time.Second):
		// No session ID for 10 seconds, give up.
		logger.Error("no session ID for 10 seconds")
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
		logger.Error("empty session ID")
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
			logger.Error("stream.Send failed setting audio format",
				"error", err)
			newSession.SessionCancel()
			return nil, err
		}
	}

	return newSession, nil
}

// CloseSession closes the specified session object and releases its resources.
func (session *SessionObject) CloseSession() {

	logger := getLogger()

	sessionId := session.SessionId

	// this logic is here to prevent a crash from closing a session twice.
	select {
	case _, ok := <-session.stopStreamingAudio:
		if ok {
			// the channel is still open, close it.
			close(session.stopStreamingAudio)
		}
	default:
		// the channel is still open, close it.
		close(session.stopStreamingAudio)
	}

	session.streamSendLock.Lock()
	err := session.SessionStream.Send(getSessionCloseRequest(""))
	session.streamSendLock.Unlock()
	if err != nil {
		logger.Error("sending session close request",
			"sessionId", sessionId,
			"error", err.Error())
	}

	// Pause to allow the close session message to be sent.
	time.Sleep(100 * time.Millisecond)

	session.SessionCancel()

	activeSessionsMap.Lock()
	delete(activeSessionsMap.OpenSessions, sessionId)
	activeSessionsMap.Unlock()

}

// HasListeningLoopExited checks if the listening loop has exited and returns
// its status along with any associated error.
func (session *SessionObject) HasListeningLoopExited() (bool, error) {

	errPtr := session.StreamLoopExitErr.Load()
	if errPtr == nil {
		return false, nil // loop hasn't exited yet
	}

	return true, *errPtr
}

// prepareInteractionCreate creates a channel in an internal map to prepare
// for an interactionCreate response. It expects the correlationId of the
// relevant interactionCreate request, which is used to route the response. It
// returns a receive-only channel, as the creating thread should not close the
// channel.
func (session *SessionObject) prepareInteractionCreate(correlationId string) (
	interactionCreateChan <-chan *api.SessionResponse, err error) {

	session.interactionCreateMapLock.Lock()
	defer session.interactionCreateMapLock.Unlock()

	// if the correlationId already has a channel, return an error
	if _, ok := session.interactionCreateMap[correlationId]; ok {
		return nil, errors.New("interaction create channel already exists")
	}

	// create the channel, put it in the map, and return it
	createChannel := make(chan *api.SessionResponse, 1)
	session.interactionCreateMap[correlationId] = createChannel
	interactionCreateChan = createChannel

	return
}

// signalInteractionCreate sends an interactionCreate response through
// the relevant channel in the interactionCreateMap. If the channel is
// not found or is empty, it returns an error.
//
// Each correlationId should only be used once, so if a map entry exists
// for the provided correlationId, it will be deleted.
func (session *SessionObject) signalInteractionCreate(correlationId string,
	response *api.SessionResponse) (err error) {

	session.interactionCreateMapLock.Lock()
	defer session.interactionCreateMapLock.Unlock()

	// attempt to fetch the channel from the map
	createChannel, ok := session.interactionCreateMap[correlationId]
	if ok == false {
		return errors.New("interaction create channel does not exist")
	}

	// an entry in the map exists. each channel should only be used once,
	// and we already have the channel, so delete the map entry.
	delete(session.interactionCreateMap, correlationId)

	// catch nil channel
	if createChannel == nil {
		return errors.New("interaction create channel is nil")
	}

	// if we got this far, the channel should be OK. Write the response.
	createChannel <- response

	return
}
