package client

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"errors"
)

// GetAudioConsumeSettings returns an api.AudioConsumeSettings object, populated
// with the specified parameters.
func (client *SdkClient) GetAudioConsumeSettings(audioChannel int32,
	audioConsumeMode api.AudioConsumeSettings_AudioConsumeMode,
	streamStartLocation api.AudioConsumeSettings_StreamStartLocation,
	startOffsetMs *api.OptionalInt32,
	audioConsumeMaxMs *api.OptionalInt32) (audioConsumeSettings *api.AudioConsumeSettings, err error) {

	if audioChannel < 0 {
		return nil, errors.New("audioChannel value must be non negative")
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

// GetNluSettings returns an api.NluSettings object
// populated with the specified parameters.
func (client *SdkClient) GetNluSettings(summarizationBulletPoints int32,
	summarizationWords int32,
	translateFromLanguage string,
	translateToLanguage string,
	enableLanguageDetect bool,
	enableTopicDetect bool,
	detectOutcomeType api.NluSettings_DetectOutcomeType,
	enableSentimentAnalysis bool,
	requestTimeoutMs *api.OptionalInt32) (nluSettings *api.NluSettings) {

	nluSettings = &api.NluSettings{
		SummarizationBulletPoints: summarizationBulletPoints,
		SummarizationWords:        summarizationWords,
		TranslateFromLanguage:     translateFromLanguage,
		TranslateToLanguage:       translateToLanguage,
		EnableLanguageDetect:      enableLanguageDetect,
		EnableTopicDetect:         enableTopicDetect,
		DetectOutcomeType:         detectOutcomeType,
		EnableSentimentAnalysis:   enableSentimentAnalysis,
		RequestTimeoutMs:          nil,
	}
	if requestTimeoutMs != nil {
		nluSettings.RequestTimeoutMs = requestTimeoutMs
	}

	return nluSettings
}

// GetNormalizationSettings returns an api.NormalizationSettings object,
// populated with the specified parameters.
func (client *SdkClient) GetNormalizationSettings(enableInverseText bool, enablePunctuationCapitalization bool,
	enableRedaction bool, enableSrtGeneration bool, enableVttGeneration bool,
	requestTimeoutMs *api.OptionalInt32) (normalizationSettings *api.NormalizationSettings) {

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
	if enableSrtGeneration {
		normalizationSettings.EnableSrtGeneration = &api.OptionalBool{Value: true}
	}
	if enableVttGeneration {
		normalizationSettings.EnableVttGeneration = &api.OptionalBool{Value: true}
	}

	return normalizationSettings
}

// GetVadSettings returns an api.VadSettings object, populated with the
// specified parameters.
func (client *SdkClient) GetVadSettings(useVad bool, bargeInTimeoutMs int32, eosDelay int32,
	endOfSpeechTimeoutMs *api.OptionalInt32,
	noiseReductionMode api.VadSettings_NoiseReductionMode,
	bargeInThreshold *api.OptionalInt32,
	snrSensitivity *api.OptionalInt32,
	streamInitDelay *api.OptionalInt32,
	volumeSensitivity *api.OptionalInt32,
	windBackMs *api.OptionalInt32,
) (vadSettings *api.VadSettings) {

	vadSettings = &api.VadSettings{
		UseVad:               &api.OptionalBool{Value: useVad},
		BargeInTimeoutMs:     &api.OptionalInt32{Value: bargeInTimeoutMs},
		EndOfSpeechTimeoutMs: endOfSpeechTimeoutMs,
		NoiseReductionMode:   noiseReductionMode,
		BargeinThreshold:     bargeInThreshold,
		EosDelayMs:           &api.OptionalInt32{Value: eosDelay},
		SnrSensitivity:       snrSensitivity,
		StreamInitDelay:      streamInitDelay,
		VolumeSensitivity:    volumeSensitivity,
		WindBackMs:           windBackMs,
	}

	return vadSettings
}

// GetAmdSettings returns an api.AmdSettings object, populated with the
// specified parameters.
func (client *SdkClient) GetAmdSettings(useAmd bool,
	amdInputText *api.OptionalString,
	faxEnable *api.OptionalBool,
	faxInputText *api.OptionalString,
	sitEnable *api.OptionalBool,
	SitReorderLocalText *api.OptionalString,
	sitVacantCodeInputText *api.OptionalString,
	sitNoCircuitLocalInputText *api.OptionalString,
	sitInterceptInputText *api.OptionalString,
	sitReorderDistantInputText *api.OptionalString,
	sitNoCircuitDistantInputText *api.OptionalString,
	sitOtherInputText *api.OptionalString,
	busyEnable *api.OptionalBool,
	BusyInputText *api.OptionalString,
	toneDetectTimeoutMs *api.OptionalInt32,
) (amdSettings *api.AmdSettings) {

	amdSettings = &api.AmdSettings{
		AmdEnable:                    &api.OptionalBool{Value: useAmd},
		AmdInputText:                 amdInputText,
		FaxEnable:                    faxEnable,
		FaxInputText:                 faxInputText,
		SitEnable:                    sitEnable,
		SitReorderLocalInputText:     SitReorderLocalText,
		SitVacantCodeInputText:       sitVacantCodeInputText,
		SitNoCircuitLocalInputText:   sitNoCircuitLocalInputText,
		SitInterceptInputText:        sitInterceptInputText,
		SitReorderDistantInputText:   sitReorderDistantInputText,
		SitNoCircuitDistantInputText: sitNoCircuitDistantInputText,
		SitOtherInputText:            sitOtherInputText,
		BusyEnable:                   busyEnable,
		BusyInputText:                BusyInputText,
		ToneDetectTimeoutMs:          toneDetectTimeoutMs,
	}

	return amdSettings
}

// GetCpaSettings returns an api.CpaSettings object, populated with the
// specified parameters.
func (client *SdkClient) GetCpaSettings(
	humanResidenceTimeMs *api.OptionalInt32,
	humanBusinessTimeMs *api.OptionalInt32,
	unknownSilenceTimeoutMs *api.OptionalInt32,
	maxTimeFromConnectMs *api.OptionalInt32,
) (cpaSettings *api.CpaSettings) {

	cpaSettings = &api.CpaSettings{
		HumanResidenceTimeMs:    humanResidenceTimeMs,
		HumanBusinessTimeMs:     humanBusinessTimeMs,
		UnknownSilenceTimeoutMs: unknownSilenceTimeoutMs,
		MaxTimeFromConnectMs:    maxTimeFromConnectMs,
	}

	return cpaSettings
}

// GetRecognitionSettings returns an api.RecognitionSettings object, populated
// with the specified parameters.
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
