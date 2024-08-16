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

var EnableVerboseLogging = false

type SessionObject struct {
	sync.RWMutex

	streamContext     context.Context
	SessionCancel     context.CancelFunc
	SessionStream     api.LumenVox_SessionClient
	sessionStreamOpen bool
	audioConfig       AudioConfig
	SessionId         string
	StreamTimeout     time.Duration
	DeploymentId      string
	OperatorId        string

	streamSendLock sync.Mutex

	audioStreamerLock        sync.Mutex
	audioStreamerInitialized bool
	audioStreamerCancel      context.CancelFunc

	audioLock      sync.Mutex
	audioForStream []byte

	ActivatedGrammarsList []string
	currentGrammarList    []*api.Grammar

	VadMessagesChannel          chan *api.VadEvent
	PartialResultsChannel       chan *api.PartialResult
	FinalResultsChannel         chan *api.FinalResult
	audioPullChannel            chan *api.AudioPullResponse
	sessionLoadChannel          chan *api.SessionLoadGrammarResponse
	SessionCloseChannel         chan struct{}
	finalResultsReadyChannel    chan bool
	createdAsrChannel           chan *api.InteractionCreateAsrResponse
	createdTranscriptionChannel chan *api.InteractionCreateTranscriptionResponse
	createdNormalizeChannel     chan *api.InteractionCreateNormalizeTextResponse
	createdGrammarParseChannel  chan *api.InteractionCreateGrammarParseResponse
	createdTtsChannel           chan *api.InteractionCreateTtsResponse
	grammarErrorChannel         chan *api.SessionEvent
	errorChan                   chan error
	stopStreamingAudio          chan bool
	audioFormatChan             chan any

	lastParseResult            *api.GrammarParseInteractionResult
	lastAsrFinalResult         *api.AsrInteractionResult
	lastAmdFinalResult         *api.AmdInteractionResult
	lastCpaFinalResult         *api.CpaInteractionResult
	receivedFinal              bool
	expectingParseResult       bool
	voiceRecognitionInProgress bool
	currentInteractionId       string
	finalizeSent               bool
	currentInteractionLanguage string

	asrInteractionsMap           map[string]*AsrInteractionObject
	transcriptionInteractionsMap map[string]*TranscriptionInteractionObject
	normalizationInteractionsMap map[string]*NormalizationInteractionObject
	ttsInteractionsMap           map[string]*TtsInteractionObject

	sessionSettingsChannel chan *api.SessionSettings
}

type SessionMapObject struct {
	sync.RWMutex
	OpenSessions map[string]*SessionObject
}

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

		VadMessagesChannel:          make(chan *api.VadEvent, 100),
		PartialResultsChannel:       make(chan *api.PartialResult, 100),
		FinalResultsChannel:         make(chan *api.FinalResult, 100),
		audioPullChannel:            make(chan *api.AudioPullResponse, 100),
		sessionLoadChannel:          make(chan *api.SessionLoadGrammarResponse, 100),
		SessionCloseChannel:         make(chan struct{}, 1),
		finalResultsReadyChannel:    make(chan bool, 100),
		createdAsrChannel:           make(chan *api.InteractionCreateAsrResponse, 100),
		createdTranscriptionChannel: make(chan *api.InteractionCreateTranscriptionResponse, 100),
		createdNormalizeChannel:     make(chan *api.InteractionCreateNormalizeTextResponse, 100),
		createdGrammarParseChannel:  make(chan *api.InteractionCreateGrammarParseResponse, 100),
		createdTtsChannel:           make(chan *api.InteractionCreateTtsResponse, 100),
		grammarErrorChannel:         make(chan *api.SessionEvent, 100),
		errorChan:                   make(chan error, 10),
		stopStreamingAudio:          make(chan bool, 1),
		audioFormatChan:             make(chan any, 1),
		sessionSettingsChannel:      make(chan *api.SessionSettings, 100),

		asrInteractionsMap:           make(map[string]*AsrInteractionObject),
		transcriptionInteractionsMap: make(map[string]*TranscriptionInteractionObject),
		normalizationInteractionsMap: make(map[string]*NormalizationInteractionObject),
		ttsInteractionsMap:           make(map[string]*TtsInteractionObject),
	}
}

func CreateNewSession(
	clientConn *grpc.ClientConn,
	streamTimeout time.Duration,
	deploymentId string,
	audioConfig AudioConfig,
	operatorId string,
) (newSession *SessionObject, err error) {

	grpcClient := api.NewLumenVoxClient(clientConn)
	streamContext, streamCancel := context.WithTimeout(context.Background(), streamTimeout)

	sessionStream, err := grpcClient.Session(streamContext)

	if err != nil {
		streamCancel()
		return nil, err
	}

	newSession = newSessionObject(streamContext, sessionStream, streamCancel, streamTimeout,
		deploymentId, operatorId, audioConfig)

	sessionIdChan := make(chan string, 1)

	go sessionResponseListener(newSession, sessionIdChan)

	newSession.streamSendLock.Lock()
	err = newSession.SessionStream.Send(getSessionCreateRequest("",
		newSession.DeploymentId, newSession.OperatorId))
	newSession.streamSendLock.Unlock()
	if err != nil {
		log.Printf("stream.Send failed sending session request: %v\n", err)
		streamCancel()
		return nil, err
	}

	var newSessionId string
	select {
	case newSessionId = <-sessionIdChan:
	case <-time.After(10 * time.Second):
		log.Printf("no session ID for 10 seconds")
		streamCancel()
		return nil, errors.New("timeout waiting for session ID")
	}

	if newSessionId != "" {
		newSession.Lock()
		newSession.SessionId = newSessionId
		newSession.Unlock()

		activeSessionsMap.Lock()
		activeSessionsMap.OpenSessions[newSessionId] = newSession
		activeSessionsMap.Unlock()
	} else {
		log.Printf("empty session ID")
		streamCancel()
		return nil, errors.New("empty session ID")
	}

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

func (session *SessionObject) CloseSession() {

	sessionId := session.SessionId

	session.streamSendLock.Lock()
	err := session.SessionStream.Send(getSessionCloseRequest(""))
	session.streamSendLock.Unlock()
	if err != nil {
		log.Printf("error sending session close request (sessionId %s): %v", sessionId, err.Error())
	}
	time.Sleep(500 * time.Millisecond)

	session.SessionCancel()

	activeSessionsMap.Lock()
	delete(activeSessionsMap.OpenSessions, sessionId)
	activeSessionsMap.Unlock()

}
