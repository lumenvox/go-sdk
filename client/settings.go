package client

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"
	"errors"
)

// GetAudioConsumeSettings returns audioConsumeSettings object populated using specified parameters
func (client *SdkClient) GetAudioConsumeSettings(audioChannel int32,
	audioConsumeMode api.AudioConsumeSettings_AudioConsumeMode,
	streamStartLocation api.AudioConsumeSettings_StreamStartLocation,
	startOffsetMs *api.OptionalInt32,
	audioConsumeMaxMs *api.OptionalInt32) (audioConsumeSettings *api.AudioConsumeSettings, err error) {

	if audioChannel < 0 {
		return nil, errors.New("audioChannel value must be valid")
	}

	audioConsumeSettings = &api.AudioConsumeSettings{
		AudioChannel:        &api.OptionalInt32{Value: audioChannel},
		AudioConsumeMode:    audioConsumeMode,
		StreamStartLocation: streamStartLocation,
		StartOffsetMs:       startOffsetMs,
		AudioConsumeMaxMs:   audioConsumeMaxMs,
	}

	return audioConsumeSettings, nil
}

// GetNormalizationSettings returns normalizationSettings object populated using specified parameters
func (client *SdkClient) GetNormalizationSettings(enableInverseText bool, enablePunctuationCapitalization bool,
	enableRedaction bool, requestTimeoutMs *api.OptionalInt32) (normalizationSettings *api.NormalizationSettings) {

	normalizationSettings = &api.NormalizationSettings{
		EnableInverseText:               nil,
		EnablePunctuationCapitalization: nil,
		EnableRedaction:                 nil,
		RequestTimeoutMs:                nil,
	}
	if enableInverseText {
		normalizationSettings.EnableInverseText = &api.OptionalBool{Value: true}
	}
	if enablePunctuationCapitalization {
		normalizationSettings.EnablePunctuationCapitalization = &api.OptionalBool{Value: true}
	}
	if enableRedaction {
		normalizationSettings.EnableRedaction = &api.OptionalBool{Value: true}
	}
	if requestTimeoutMs != nil {
		normalizationSettings.RequestTimeoutMs = requestTimeoutMs
	}

	return normalizationSettings
}

// GetVadSettings returns vadSettings object populated using specified parameters
func (client *SdkClient) GetVadSettings(useVad bool, bargeInTimeout int32, eosDelay int32,
	endOfSpeechTimeoutMs *api.OptionalInt32,
	noiseReductionMode api.VadSettings_NoiseReductionMode,
	bargeInThreshold *api.OptionalInt32,
	snrSensitivity *api.OptionalInt32,
	streamInitDelay *api.OptionalInt32,
	volumeSensitivity *api.OptionalInt32,
	windBackMs *api.OptionalInt32,
) (vadSettings *api.VadSettings) {

	vadSettings = &api.VadSettings{
		EndOfSpeechTimeoutMs: endOfSpeechTimeoutMs,
		NoiseReductionMode:   noiseReductionMode,
		BargeinThreshold:     bargeInThreshold,
		SnrSensitivity:    snrSensitivity,
		StreamInitDelay:   streamInitDelay,
		VolumeSensitivity: volumeSensitivity,
		WindBackMs:        windBackMs,
	}
	vadSettings.UseVad = &api.OptionalBool{Value: useVad}
	vadSettings.BargeInTimeoutMs = &api.OptionalInt32{Value: bargeInTimeout}
	vadSettings.EosDelayMs = &api.OptionalInt32{Value: eosDelay}

	return vadSettings
}

// GetRecognitionSettings returns recognitionSettings object populated using specified parameters
func (client *SdkClient) GetRecognitionSettings(decodeTimeout int32, enablePartialResults bool,
	maxAlternatives *api.OptionalInt32,
	trimSilence *api.OptionalInt32,
	confidenceThreshold *api.OptionalInt32,
) (recognitionSettings *api.RecognitionSettings) {

	recognitionSettings = &api.RecognitionSettings{
		MaxAlternatives:      maxAlternatives,
		TrimSilenceValue:     trimSilence,
		EnablePartialResults: &api.OptionalBool{Value: enablePartialResults},
		ConfidenceThreshold:  confidenceThreshold,
		DecodeTimeout:        &api.OptionalInt32{Value: decodeTimeout},
	}

	return recognitionSettings
}
