package session

import (
	"context"
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"errors"
	"log"
	"time"
)

var (
	defaultBatchChunkSize       int = 10000
	defaultBatchChunkDelayMs    int = 25
	defaultStreamingChunkSizeMs int = 20
)

type AudioConfig struct {
	Format     api.AudioFormat_StandardAudioFormat
	SampleRate int

	IsBatch bool

	BatchChunkSize    int
	BatchChunkDelayMs int

	StreamingChunkSizeMs int
}

func (session *SessionObject) AddAudio(audioData []byte) error {

	sessionDeadline, ok := session.streamContext.Deadline()
	if ok != true {
		return errors.New("failed to retrieve session stream deadline")
	}

	producerCtx, producerCancel := context.WithDeadline(context.Background(), sessionDeadline)

	if session.audioConfig.IsBatch {
		StreamBatchAudio(session, producerCtx, audioData)
		producerCancel()
	} else {
		err := StreamStreamAudio(session, producerCtx, producerCancel, audioData)
		if err != nil {
			return err
		}
	}

	return nil
}

func StreamBatchAudio(session *SessionObject, producerCtx context.Context, audioData []byte) {

	if len(audioData) == 0 {
		return
	}

	chunkSize := session.audioConfig.BatchChunkSize
	chunkDelayMs := session.audioConfig.BatchChunkDelayMs

	if chunkSize == 0 {
		chunkSize = defaultBatchChunkSize
	}
	if chunkDelayMs == 0 {
		chunkDelayMs = defaultBatchChunkDelayMs
	}

	firstChunkSize := chunkSize
	if len(audioData) < chunkSize {
		firstChunkSize = len(audioData)
	}
	session.streamSendLock.Lock()
	err := session.SessionStream.Send(getAudioPushRequest("", audioData[:firstChunkSize]))
	session.streamSendLock.Unlock()
	if err != nil {
		log.Printf("error sending audio to API: %+v", err)
	}
	remainingAudioData := audioData[firstChunkSize:]

	if len(remainingAudioData) == 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(chunkDelayMs) * time.Millisecond)

audioStreamLoop:
	for len(remainingAudioData) > 0 {
		select {
		case <-ticker.C:

			nextChunkSize := chunkSize
			if len(remainingAudioData) < chunkSize {
				nextChunkSize = len(remainingAudioData)
			}

			session.streamSendLock.Lock()
			err := session.SessionStream.Send(getAudioPushRequest("", remainingAudioData[:nextChunkSize]))
			session.streamSendLock.Unlock()
			if err != nil {
				log.Printf("error sending audio to API: %+v", err)
			}

			remainingAudioData = remainingAudioData[firstChunkSize:]

		case <-session.stopStreamingAudio:
			break audioStreamLoop

		case <-producerCtx.Done():
			break audioStreamLoop
		}
	}

	ticker.Stop()
}

func (session *SessionObject) handleInternalStream(producerCtx context.Context, chunkSizeMs, chunkSizeBytes int) {


	ticker := time.NewTicker(time.Duration(chunkSizeMs) * time.Millisecond)

audioStreamLoop:
	for {
		select {
		case <-ticker.C:

			session.audioLock.Lock()
			availableDataLen := len(session.audioForStream)
			if availableDataLen == 0 {
				session.audioLock.Unlock()
				break
			}

			nextChunkSizeBytes := chunkSizeBytes
			if availableDataLen < nextChunkSizeBytes {
				nextChunkSizeBytes = availableDataLen
			}

			nextChunkData := session.audioForStream[:nextChunkSizeBytes]
			session.audioForStream = session.audioForStream[nextChunkSizeBytes:]

			session.audioLock.Unlock()

			session.streamSendLock.Lock()
			err := session.SessionStream.Send(getAudioPushRequest("", nextChunkData))
			session.streamSendLock.Unlock()
			if err != nil {
				log.Printf("error sending audio to API: %+v", err)
			}

		case <-session.stopStreamingAudio:
			break audioStreamLoop

		case <-producerCtx.Done():
			break audioStreamLoop
		}
	}

	ticker.Stop()
}

func StreamStreamAudio(session *SessionObject, producerCtx context.Context, producerCancel context.CancelFunc,
	audioData []byte) error {

	if len(audioData) == 0 {
		return nil
	}

	session.audioStreamerLock.Lock()
	if session.audioStreamerInitialized == false {

		chunkSizeMs := session.audioConfig.StreamingChunkSizeMs
		if chunkSizeMs == 0 {
			chunkSizeMs = defaultStreamingChunkSizeMs
		}

		sampleRate := session.audioConfig.SampleRate
		if sampleRate <= 0 {
			return errors.New("invalid sample rate")
		}
		format := session.audioConfig.Format
		if format == api.AudioFormat_STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE {
			return errors.New("invalid format NO_AUDIO_RESOURCE")
		}
		samplesPerChunk := sampleRate * chunkSizeMs / 1000

		chunkSizeBytes := samplesPerChunk
		switch format {
		case api.AudioFormat_STANDARD_AUDIO_FORMAT_LINEAR16:
			chunkSizeBytes *= 2
		case api.AudioFormat_STANDARD_AUDIO_FORMAT_ULAW:
		case api.AudioFormat_STANDARD_AUDIO_FORMAT_ALAW:
		default:
			return errors.New("unsupported audio format")
		}

		go session.handleInternalStream(producerCtx, chunkSizeMs, chunkSizeBytes)
		session.audioStreamerInitialized = true
		session.audioStreamerCancel = producerCancel
	}
	session.audioStreamerLock.Unlock()

	session.audioLock.Lock()
	session.audioForStream = append(session.audioForStream, audioData...)
	session.audioLock.Unlock()

	return nil
}
