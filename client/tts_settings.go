package client

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"log"
)

// GetTtsInlineSynthesisSettings returns an api.TtsInlineSynthesisSettings
// object, populated with the specified parameters.
func (client *SdkClient) GetTtsInlineSynthesisSettings(
	voiceName string,
	voiceGender string,
	voiceEmphasis string,
	voicePitch string,
	voiceRate string,
	voiceVolume string,
) (inlineSettings *api.TtsInlineSynthesisSettings) {

	synthSettings := &api.TtsInlineSynthesisSettings{
		Voice:                nil,
		SynthEmphasisLevel:   nil,
		SynthProsodyPitch:    nil,
		SynthProsodyContour:  nil, // not currently supported
		SynthProsodyRate:     nil,
		SynthProsodyDuration: nil, // not currently supported
		SynthProsodyVolume:   nil,
		SynthVoiceAge:        nil, // not currently supported
		SynthVoiceGender:     nil,
	}
	if voiceName != "" {
		synthSettings.Voice = &api.OptionalString{Value: voiceName}
	} else {
		// Only assign gender if the voice is not specified.
		if voiceGender != "" && voiceGender != "neutral" {
			if voiceGender == "male" || voiceGender == "female" {
				synthSettings.SynthVoiceGender = &api.OptionalString{Value: voiceGender}
			} else {
				// Some unsupported value. Warn, but do not return an error.
				log.Printf("warning: invalid voiceGender \"%s\" (should be \"neutral\", \"male\", or \"female\")", voiceGender)
			}
		}
	}

	if voiceEmphasis != "" {
		synthSettings.SynthEmphasisLevel = &api.OptionalString{Value: voiceEmphasis}
	}
	if voicePitch != "" {
		synthSettings.SynthProsodyPitch = &api.OptionalString{Value: voicePitch}
	}
	if voiceRate != "" {
		synthSettings.SynthProsodyRate = &api.OptionalString{Value: voiceRate}
	}
	if voiceVolume != "" {
		synthSettings.SynthProsodyVolume = &api.OptionalString{Value: voiceVolume}
	}

	return synthSettings
}
