package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"context"
	"errors"
	"time"
)

var (
	defaultBatchChunkSize       = 10000
	defaultBatchChunkDelayMs    = 25
	defaultStreamingChunkSizeMs = 20
)

// AudioConfig contains the configuration for audio streamed to the API. It
// includes things like the audio format, sample rate, and streaming/batch
// configuration.
type AudioConfig struct {
	Format     api.AudioFormat_StandardAudioFormat
	SampleRate int

	IsBatch bool

	BatchChunkSize    int // chunk size (bytes) for batch audio
	BatchChunkDelayMs int // inter-chunk delay (ms) for batch audio

	// parameters for streaming audio push
	StreamingChunkSizeMs int // chunk size (ms) for streaming audio
}

// AddAudio sends the specified audio data through the session stream. This function
// differs in behavior based on the session's AudioConfig object. If the config indicates
// batch mode is active, this function will block until all audio has been sent. If the
// config indicates streaming mode, the audio will be added to an internal queue which
// will be streamed in the background by a separate goroutine.
func (session *SessionObject) AddAudio(audioData []byte) error {

	// Get the deadline for the parent session
	sessionDeadline, ok := session.streamContext.Deadline()
	if ok != true {
		return errors.New("failed to retrieve session stream deadline")
	}

	// Set up a producer context with the same deadline as the session
	producerCtx, producerCancel := context.WithDeadline(context.Background(), sessionDeadline)

	// Check audio config: streaming or batch?
	if session.audioConfig.IsBatch {
		// if we're in batch mode, block until all audio is sent
		streamBatchAudio(session, producerCtx, audioData)
		producerCancel()
	} else {
		// if we're in streaming mode, add the audio to the internal queue
		err := streamStreamAudio(session, producerCtx, producerCancel, audioData)
		if err != nil {
			return err
		}
	}

	return nil
}

// streamBatchAudio sends batch audio data into the session. It automatically chunks the data
// and adds delays based on the session audioConfig.
func streamBatchAudio(session *SessionObject, producerCtx context.Context, audioData []byte) {

	// Immediately return if there is no data to stream
	if len(audioData) == 0 {
		return
	}

	logger := getLogger()

	// Get audio chunk parameters from the audio config
	chunkSize := session.audioConfig.BatchChunkSize
	chunkDelayMs := session.audioConfig.BatchChunkDelayMs

	// Catch invalid chunk parameters
	if chunkSize < 0 {
		logger.Warn("invalid BatchChunkSize (should be non-negative) - using default",
			"chunkSpecified", chunkSize,
			"default", defaultBatchChunkSize)
		chunkSize = 0 // value assigned below
	}
	if chunkDelayMs < 0 {
		logger.Warn("invalid BatchChunkDelayMs (should be non-negative) - using default",
			"chunkDelayMsSpecified", chunkDelayMs,
			"default", defaultBatchChunkDelayMs)
		chunkDelayMs = 0 // value assigned below
	}

	// Use defaults if necessary
	if chunkSize == 0 {
		chunkSize = defaultBatchChunkSize
	}
	if chunkDelayMs == 0 {
		chunkDelayMs = defaultBatchChunkDelayMs
	}

	// Send the first chunk of audio into the session
	firstChunkSize := chunkSize
	if len(audioData) < chunkSize {
		firstChunkSize = len(audioData)
	}
	session.streamSendLock.Lock()
	err := session.SessionStream.Send(getAudioPushRequest("", audioData[:firstChunkSize]))
	session.streamSendLock.Unlock()
	if err != nil {
		logger.Error("sending audio to API",
			"error", err)
	}
	remainingAudioData := audioData[firstChunkSize:]

	// if there's no data left to send, return
	if len(remainingAudioData) == 0 {
		return
	}

	// if we reach this point, there's still data left to send. initialize a ticker and start
	// a loop to send any remaining chunks of audio.
	ticker := time.NewTicker(time.Duration(chunkDelayMs) * time.Millisecond)

audioStreamLoop:
	for len(remainingAudioData) > 0 {
		select {
		case <-ticker.C:
			// One more tick, send the next chunk and update the remaining audio

			// Determine the chunk size
			nextChunkSize := chunkSize
			if len(remainingAudioData) < chunkSize {
				nextChunkSize = len(remainingAudioData)
			}

			// Send the chunk
			session.streamSendLock.Lock()
			err := session.SessionStream.Send(getAudioPushRequest("", remainingAudioData[:nextChunkSize]))
			session.streamSendLock.Unlock()
			if err != nil {
				logger.Error("sending audio to API",
					"error", err)
			}

			// Update the remaining audio
			remainingAudioData = remainingAudioData[nextChunkSize:]

		case <-session.stopStreamingAudio:
			// Session closing, stop streaming
			break audioStreamLoop

		case <-producerCtx.Done():
			// Producer context canceled or timed out, stop streaming
			break audioStreamLoop
		}
	}

	// Done streaming. Stop the ticker and return.
	ticker.Stop()
}

// handleInternalStream streams audio in the background. It is typically run in
// its own goroutine.
func (session *SessionObject) handleInternalStream(producerCtx context.Context, chunkSizeMs, chunkSizeBytes int) {

	// Set up a ticker to handle audio streaming
	ticker := time.NewTicker(time.Duration(chunkSizeMs) * time.Millisecond)

	logger := getLogger()

audioStreamLoop:
	for {
		select {
		case <-ticker.C:

			// One more tick, check for any new data.
			session.audioLock.Lock()
			availableDataLen := len(session.audioForStream)
			if availableDataLen == 0 {
				session.audioLock.Unlock()
				break
			}

			// Determine how much data to get.
			nextChunkSizeBytes := chunkSizeBytes
			if availableDataLen < nextChunkSizeBytes {
				nextChunkSizeBytes = availableDataLen
			}

			// Save the data and remove it from the buffer
			nextChunkData := session.audioForStream[:nextChunkSizeBytes]
			session.audioForStream = session.audioForStream[nextChunkSizeBytes:]

			// Now that we've done everything with the internal buffer, release the lock.
			session.audioLock.Unlock()

			// Send the chunk
			session.streamSendLock.Lock()
			err := session.SessionStream.Send(getAudioPushRequest("", nextChunkData))
			session.streamSendLock.Unlock()
			if err != nil {
				logger.Error("sending audio to API",
					"error", err)
			}

		case <-session.stopStreamingAudio:
			// Session closing, stop streaming.
			break audioStreamLoop

		case <-producerCtx.Done():
			// Producer context canceled or timed out, stop streaming
			break audioStreamLoop
		}
	}

	// Done streaming. stop the ticker and return.
	ticker.Stop()
}

// streamStreamAudio is used to add audio for the session to stream. Audio data
// is written to a buffer in the session object. A session-specific goroutine
// reads audio from this buffer and streams it in the background according to
// the audio configuration.
//
// On the first call to this function, the goroutine is initialized. On later
// calls, this function is only responsible for adding audio to the internal
// session buffer as the internal streamer continues to stream.
func streamStreamAudio(session *SessionObject, producerCtx context.Context, producerCancel context.CancelFunc,
	audioData []byte) error {

	// Immediately return if there is no data to stream.
	if len(audioData) == 0 {
		// no need to signal an error - leave that for real errors.
		return nil
	}

	logger := getLogger()

	// Check if we have initialized the audio streaming goroutine. If we haven't,
	// initialize it.
	session.audioStreamerLock.Lock()
	if session.audioStreamerInitialized == false {

		// Get the chunk size in milliseconds
		chunkSizeMs := session.audioConfig.StreamingChunkSizeMs
		if chunkSizeMs < 0 {
			logger.Warn("invalid StreamingChunkSizeMs (should be non-negative) - using default",
				"chunkSizeMs", chunkSizeMs,
				"default", defaultStreamingChunkSizeMs)
			chunkSizeMs = 0
		}
		if chunkSizeMs == 0 {
			chunkSizeMs = defaultStreamingChunkSizeMs
		}

		// Based on the milliseconds, format, and sample rate, get the chunk size in bytes
		sampleRate := session.audioConfig.SampleRate
		if sampleRate <= 0 {
			return errors.New("invalid sample rate (must be positive)")
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

	// Add audio to the session buffer. The internal streamer will read from this buffer,
	// automatically handling chunking and delays.
	session.audioLock.Lock()
	session.audioForStream = append(session.audioForStream, audioData...)
	session.audioLock.Unlock()

	return nil
}

// AudioBufferSize returns the size of the internal audio buffer. This can be used to
// monitor streaming operations.
func (session *SessionObject) AudioBufferSize() int {
	return len(session.audioForStream)
}

// ClearAudioBuffer clears the internal audio queue, preventing any new
// packets from being sent. This is only relevant for streaming operations.
func (session *SessionObject) ClearAudioBuffer() {
	session.audioLock.Lock()
	session.audioForStream = nil
	session.audioLock.Unlock()
}
