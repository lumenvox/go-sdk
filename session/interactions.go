package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lumenvox/protos-go/lumenvox/api"
)

// vadInteractionRecord records the data from a single VAD interaction.
// For a successful interaction, it will store the values for barge-in and
// barge-out. If a timeout was received instead, the struct values will
// indicate that as well.
type vadInteractionRecord struct {
	beginProcessingReceived bool
	bargeInTimeoutReceived  bool
	bargeInReceived         int
	bargeOutTimeoutReceived bool
	bargeOutReceived        int
}

func createEmptyVadInteractionRecord() *vadInteractionRecord {
	return &vadInteractionRecord{
		beginProcessingReceived: false,
		bargeInTimeoutReceived:  false,
		bargeInReceived:         0,
		bargeOutReceived:        0,
	}
}

// interactionFinalizeHelper is used to coordinate finalize interaction requests and responses
type interactionFinalizeHelper struct {
	responseChannel chan error
	deadline        time.Time
}

// prepareInteractionFinalize creates a helper in an internal map to prepare
// for an interaction finalize response. It expects the correlationId of the
// relevant finalize request, which is used to route the response, as
// well as a deadline to receive the response. It returns a receive-only channel,
// as the creating thread should not close the channel.
func (session *SessionObject) prepareInteractionFinalize(correlationId string, deadline time.Duration) (
	finalizeResponseChan <-chan error, err error) {

	session.interactionFinalizeMapLock.Lock()
	defer session.interactionFinalizeMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.interactionFinalizeMap[correlationId]; ok {
		return nil, errors.New("interaction finalize channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	finalizeHelper := &interactionFinalizeHelper{
		responseChannel: make(chan error, 1),
		deadline:        time.Now().Add(deadline),
	}
	session.interactionFinalizeMap[correlationId] = finalizeHelper

	finalizeResponseChan = finalizeHelper.responseChannel

	return
}

type sessionSettingsResult struct {
	SessionSettings *api.SessionSettings
	Error           error
}

type sessionGetSettingsHelper struct {
	sessionSettingsChannel chan *sessionSettingsResult
	deadline               time.Time
}

// prepareGetSessionSettings creates a channel in an internal map to prepare
// for a get session settings request. It expects the correlationId of the
// relevant get session settings request, which is used to route the response.
// It returns a receive-only channel, as the creating thread should not close the
// channel.
func (session *SessionObject) prepareGetSessionSettings(correlationId string, deadline time.Duration) (
	sessionSettingsChan <-chan *sessionSettingsResult, err error) {

	session.sessionGetSettingsMapLock.Lock()
	defer session.sessionGetSettingsMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.sessionGetSettingsMap[correlationId]; ok {
		return nil, errors.New("session get settings channel already exists")
	}

	// create the helper, put it in the map, and return it
	settingsHelper := &sessionGetSettingsHelper{
		sessionSettingsChannel: make(chan *sessionSettingsResult, 1),
		deadline:               time.Now().Add(deadline),
	}
	session.sessionGetSettingsMap[correlationId] = settingsHelper
	sessionSettingsChan = settingsHelper.sessionSettingsChannel

	return
}

type sessionSetSettingsHelper struct {
	responseChannel chan error
	deadline        time.Time
}

// prepareSetSessionSettings creates a helper in an internal map to prepare
// for a set session settings request. It expects the correlationId of the
// relevant set session settings request, which is used to route the response.
// It returns a receive-only channel, as the creating thread should not close the
// channel.
func (session *SessionObject) prepareSetSessionSettings(correlationId string, deadline time.Duration) (
	responseChan <-chan error, err error) {

	session.sessionSetSettingsMapLock.Lock()
	defer session.sessionSetSettingsMapLock.Unlock()

	// if the correlationId is already in the map, return an error
	if _, ok := session.sessionSetSettingsMap[correlationId]; ok {
		return nil, errors.New("session set settings channel already exists")
	}

	// create the helper, put it in the map, and return the channel
	settingsHelper := &sessionSetSettingsHelper{
		responseChannel: make(chan error, 1),
		deadline:        time.Now().Add(deadline),
	}
	session.sessionSetSettingsMap[correlationId] = settingsHelper
	responseChan = settingsHelper.responseChannel

	return
}

// PullTtsAudio fetches generated audio from a specified TTS interaction.
func (session *SessionObject) PullTtsAudio(interactionId string, audioChannel int32, audioStartMs int32,
	audioLengthMs int32) (audioData []byte, err error) {

	logger := getLogger()

	// Set up a channel to track the returned audio data
	correlationId := uuid.NewString()
	audioPullChannel, err := session.prepareAudioPull(correlationId)
	if err != nil {
		logger.Error(err.Error(),
			"correlation_id", correlationId,
			"interaction_id", interactionId,
			"session_id", session.SessionId)
		return nil, err
	}
	// make sure the channel is removed from the map if we exit early (send
	// failure or timeout). this is a no-op if a response handler already removed it.
	defer session.cleanupAudioPull(correlationId)

	// Send audio pull request using the provided interactionId, adding specified parameters
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getAudioPullRequest(correlationId, interactionId, audioChannel, audioStartMs, audioLengthMs))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending AudioPullRequest error: %v", err)
		logger.Error("sending audio pull request",
			"error", err.Error())
		return nil, err
	}

	var pullResult *audioPullResult
	isFinalDataChunk := false
	for isFinalDataChunk == false {
		select {
		case pullResult = <-audioPullChannel:
		case <-time.After(5 * time.Second):
			// timed out - log error and return any data received so far
			logger.Error("timed out waiting for final audio chunk",
				"interaction_id", interactionId,
				"audio_channel", audioChannel)
			return audioData, TimeoutError
		}

		if pullResult == nil {
			// This should not happen, but if it does, just continue the loop
			continue
		} else if pullResult.Error != nil {
			// the API reported an error - return it along with any data received so far
			return audioData, pullResult.Error
		} else if pullResult.Response != nil {
			isFinalDataChunk = pullResult.Response.FinalDataChunk
			audioData = append(audioData, pullResult.Response.AudioData...)
		}
	}

	return audioData, nil
}

func (session *SessionObject) BeginProcessingInteraction(interactionId string) (err error) {

	// validate the request
	if interactionId == "" {
		return errors.New("empty interaction id")
	}

	// assuming the request is valid, send the begin-processing request
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getInteractionBeginProcessingRequest("", interactionId))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionFinalizeProcessingRequest error: %v", err)
		logger := getLogger()
		logger.Error("sending finalize processing request",
			"error", err.Error())
		return err
	}

	return nil
}

// FinalizeInteraction attempts to finalize an existing interaction. This is not limited to a
// single interaction type.
func (session *SessionObject) FinalizeInteraction(interactionId string) (err error) {

	// validate the request
	if interactionId == "" {
		return errors.New("empty interaction id")
	}

	// generate correlation ID and prepare for response coordination
	correlationId := uuid.NewString()
	finalizeResponseChan, err := session.prepareInteractionFinalize(correlationId, 10*time.Second)
	if err != nil {
		logger := getLogger()
		logger.Error("preparing finalize coordination",
			"error", err.Error(),
			"interaction_id", interactionId)
		return err
	}

	// Send finalize request using the provided interactionId and correlationId
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getInteractionFinalizeRequest(correlationId, interactionId))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionFinalizeProcessingRequest error: %v", err)
		logger := getLogger()
		logger.Error("sending finalize processing request",
			"error", err.Error())
		return err
	}

	// Wait for response
	select {
	case responseErr := <-finalizeResponseChan:
		return responseErr
	case <-time.After(10 * time.Second):
		logger := getLogger()
		logger.Error("timed out waiting for finalize response",
			"interaction_id", interactionId,
			"correlation_id", correlationId)
		return TimeoutError
	}
}

// GetSessionSettings fetches the current session settings from the API.
func (session *SessionObject) GetSessionSettings() (settings *api.SessionSettings, err error) {

	logger := getLogger()

	// generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// create a helper struct to wait for the response
	sessionSettingsChannel, err := session.prepareGetSessionSettings(correlationId, 10*time.Second)
	if err != nil {
		logger.Error("preparing get session settings coordination",
			"error", err,
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, err
	}

	// no errors during preparation, send the get session settings request
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getGetSessionSettingsRequest(correlationId))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending SessionGetSettingsRequest error: %v", err)
		logger.Error("sending session get settings request",
			"error", err.Error())
		return nil, err
	}

	// wait for the response
	select {
	case result := <-sessionSettingsChannel:
		return result.SessionSettings, result.Error
	case <-time.After(10 * time.Second):
		logger.Error("timed out waiting for session settings response",
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return nil, TimeoutError
	}
}

func (session *SessionObject) SetSessionSettings(settings *api.SessionSettings) (err error) {

	logger := getLogger()

	// generate a correlation ID to track the response when it arrives
	correlationId := uuid.NewString()

	// create a helper struct to wait for the response
	sessionSettingsChannel, err := session.prepareSetSessionSettings(correlationId, 10*time.Second)
	if err != nil {
		logger.Error("preparing set session settings coordination",
			"error", err,
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return err
	}

	// no errors during preparation, send the set session settings request
	session.streamSendLock.Lock()
	err = session.sessionStream.Send(getSetSessionSettingsRequest(correlationId, settings))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending SessionSetSettingsRequest error: %v", err)
		logger.Error("sending session set settings request",
			"error", err.Error())
		return err
	}

	// wait for the response
	select {
	case responseErr := <-sessionSettingsChannel:
		return responseErr
	case <-time.After(10 * time.Second):
		logger.Error("timed out waiting for set session settings response",
			"correlation_id", correlationId,
			"session_id", session.SessionId)
		return TimeoutError
	}
}
