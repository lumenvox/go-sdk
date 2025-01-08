package session

import (
	"fmt"
	"log"
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
