package client

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"log"
)

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
		SynthProsodyContour:  nil,
		SynthProsodyRate:     nil,
		SynthProsodyDuration: nil,
		SynthProsodyVolume:   nil,
		SynthVoiceAge:        nil,
		SynthVoiceGender:     nil,
	}
	if voiceName != "" {
		synthSettings.Voice = &api.OptionalString{Value: voiceName}
	} else {
		if voiceGender != "" && voiceGender != "neutral" {
			if voiceGender == "male" || voiceGender == "female" {
				synthSettings.SynthVoiceGender = &api.OptionalString{Value: voiceGender}
			} else {
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
