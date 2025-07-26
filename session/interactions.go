package session

import (
	"github.com/lumenvox/protos-go/lumenvox/api"

	"fmt"
	"github.com/google/uuid"
	"time"
)

// vadInteractionRecord records the data from a single VAD interaction.
// For a successful interaction, it will store the values for barge-in and
// barge-out. If a timeout was received instead, the struct values will
// indicate that as well.
type vadInteractionRecord struct {
	beginProcessingReceived bool
	bargeInTimeoutReceived  bool
	bargeInReceived         int
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

// PullTtsAudio fetches generated audio from a specified TTS interaction.
func (session *SessionObject) PullTtsAudio(interactionId string, audioChannel int32, audioStartMs int32,
	audioLengthMs int32) (audioData []byte, err error) {

	logger := getLogger()

	// Set up a channel to track the returned audio data
	correlationId := uuid.NewString()
	audioPullChannel, err := session.prepareAudioPull(correlationId)
	if err != nil {
		logger.Error(err.Error(),
			"correlationId", correlationId,
			"interactionId", interactionId,
			"sessionId", session.SessionId)
		return nil, err
	}

	// Send audio pull request using the provided interactionId, adding specified parameters
	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getAudioPullRequest(correlationId, interactionId, audioChannel, audioStartMs, audioLengthMs))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending AudioPullRequest error: %v", err)
		logger.Error("sending audio pull request",
			"error", err.Error())
		return nil, err
	}

	var audioPullResponse *api.AudioPullResponse
	isFinalDataChunk := false
	for isFinalDataChunk == false {
		select {
		case audioPullResponse = <-audioPullChannel:
		case <-time.After(5 * time.Second):
			// timed out - log error and return any data received so far
			logger.Error("timed out waiting for final audio chunk",
				"interactionId", interactionId,
				"audioChannel", audioChannel)
			return audioData, TimeoutError
		}

		if audioPullResponse == nil {
			// This should not happen, but if it does, just continue the loop
			continue
		} else {
			isFinalDataChunk = audioPullResponse.FinalDataChunk
			audioData = append(audioData, audioPullResponse.AudioData...)
		}
	}

	return audioData, nil
}

// FinalizeInteraction attempts to finalize an existing interaction. This is not limited to a
// single interaction type.
func (session *SessionObject) FinalizeInteraction(interactionId string) (err error) {

	logger := getLogger()

	// Send finalize request using the provided interactionId, adding specified parameters

	session.streamSendLock.Lock()
	err = session.SessionStream.Send(getInteractionFinalizeRequest("", interactionId))
	session.streamSendLock.Unlock()
	if err != nil {
		session.errorChan <- fmt.Errorf("sending InteractionFinalizeProcessingRequest error: %v", err)
		logger.Error("sending finalize processing request",
			"error", err.Error())
		return err
	}

	return nil
}
