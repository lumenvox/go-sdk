package session

import (
	"sync"

	"github.com/lumenvox/protos-go/lumenvox/api"
)

// vadSignal records the first occurrence of a VAD event for a single-utterance
// interaction (ASR, AMD, CPA) and wakes every goroutine waiting on it.
//
// The server is expected to send each of these events at most once per
// interaction. If it repeats one, the repeat is ignored, and the value from
// the first occurrence is kept.
type vadSignal struct {
	once sync.Once
	// done is closed when the event arrives.
	done chan struct{}
	// value holds the event's audio offset in milliseconds, when it has one.
	// It is written before done is closed, so it is safe to read once done is
	// closed.
	value int
}

func newVadSignal() *vadSignal {
	return &vadSignal{done: make(chan struct{})}
}

// fire records the event and wakes all waiters. It returns false if the event
// had already fired, in which case value is discarded.
func (signal *vadSignal) fire(value int) (first bool) {

	signal.once.Do(func() {
		signal.value = value
		close(signal.done)
		first = true
	})

	return first
}

// fired reports whether the event has arrived, without blocking.
func (signal *vadSignal) fired() bool {

	select {
	case <-signal.done:
		return true
	default:
		return false
	}
}

// vadSignals holds the VAD signals of an interaction type that handles a single
// utterance (ASR, AMD, CPA).
type vadSignals struct {
	vadBeginProcessing *vadSignal
	vadBargeIn         *vadSignal
	vadBargeOut        *vadSignal
	vadBargeInTimeout  *vadSignal
	vadBargeOutTimeout *vadSignal
}

func newVadSignals() vadSignals {
	return vadSignals{
		vadBeginProcessing: newVadSignal(),
		vadBargeIn:         newVadSignal(),
		vadBargeOut:        newVadSignal(),
		vadBargeInTimeout:  newVadSignal(),
		vadBargeOutTimeout: newVadSignal(),
	}
}

// handleVadEvent fires the signal matching the event type. It runs in the
// session response listener, so it must never block.
func (signals *vadSignals) handleVadEvent(vadEvent *api.VadEvent) {

	logger := getLogger()

	var signal *vadSignal
	value := 0

	switch vadEvent.GetVadEventType() {
	case api.VadEvent_VAD_EVENT_TYPE_BEGIN_PROCESSING:
		signal = signals.vadBeginProcessing
	case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN:
		signal = signals.vadBargeIn
		value = vadEventAudioOffset(vadEvent)
	case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH:
		signal = signals.vadBargeOut
		value = vadEventAudioOffset(vadEvent)
	case api.VadEvent_VAD_EVENT_TYPE_BARGE_IN_TIMEOUT:
		signal = signals.vadBargeInTimeout
	case api.VadEvent_VAD_EVENT_TYPE_END_OF_SPEECH_TIMEOUT:
		signal = signals.vadBargeOutTimeout
	default:
		logger.Warn("unexpected VAD event",
			"interaction_id", vadEvent.GetInteractionId(),
			"to_state", vadEvent.GetVadEventType().String())
		return
	}

	if !signal.fire(value) {
		logger.Warn("ignoring duplicate VAD event",
			"interaction_id", vadEvent.GetInteractionId(),
			"type", vadEvent.GetVadEventType().String())
	}
}

// vadEventAudioOffset returns the event's audio offset in milliseconds, or 0
// if the event has none.
func vadEventAudioOffset(vadEvent *api.VadEvent) int {

	audioOffset := vadEvent.GetAudioOffset()
	if audioOffset == nil {
		getLogger().Warn("VAD event has empty AudioOffset, using 0",
			"interaction_id", vadEvent.GetInteractionId(),
			"type", vadEvent.GetVadEventType().String())
		return 0
	}

	return int(audioOffset.GetValue())
}
