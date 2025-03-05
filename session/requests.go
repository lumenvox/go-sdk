package session

import (
	"github.com/lumenvox/go-sdk/lumenvox/api"

	"github.com/google/uuid"
)

// getSessionCreateRequest returns a create session request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getSessionCreateRequest(correlationId string, deploymentId string, operatorId string) (sessionRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	sessionRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_SessionRequest{
			SessionRequest: &api.SessionRequestMessage{
				Name: &api.SessionRequestMessage_SessionCreate{
					SessionCreate: &api.SessionCreateRequest{
						DeploymentId: deploymentId,
						OperatorId:   operatorId,
					},
				},
			},
		},
	}

	return sessionRequest
}

// getAudioFormatRequest returns an audio format request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getAudioFormatRequest(correlationId string, audioConfig AudioConfig) (formatRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	formatRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_SessionRequest{
			SessionRequest: &api.SessionRequestMessage{
				Name: &api.SessionRequestMessage_SessionAudioFormat{
					SessionAudioFormat: &api.SessionInboundAudioFormatRequest{
						AudioFormat: &api.AudioFormat{
							StandardAudioFormat: audioConfig.Format,
							SampleRateHertz:     &api.OptionalInt32{Value: int32(audioConfig.SampleRate)},
						},
					},
				},
			},
		},
	}

	return formatRequest
}

// getAsrRequest returns an ASR interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getAsrRequest(
	correlationId string,
	language string,
	grammars []*api.Grammar,
	grammarSettings *api.GrammarSettings,
	recognitionSettings *api.RecognitionSettings,
	vadSettings *api.VadSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (asrRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	asrRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateAsr{
					InteractionCreateAsr: &api.InteractionCreateAsrRequest{
						Language:                   language,
						Grammars:                   grammars,
						GrammarSettings:            grammarSettings,
						RecognitionSettings:        recognitionSettings,
						VadSettings:                vadSettings,
						AudioConsumeSettings:       audioConsumeSettings,
						GeneralInteractionSettings: generalInteractionSettings,
					},
				},
			},
		},
	}

	return asrRequest
}

// getTranscriptionRequest returns a transcription interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be auto-generated.
func getTranscriptionRequest(
	correlationId string,
	language string,
	embeddedGrammars []*api.Grammar,
	vadSettings *api.VadSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	normalizationSettings *api.NormalizationSettings,
	recognitionSettings *api.RecognitionSettings,
	languageModelName string,
	acousticModelName string,
	enablePostProcessing string,
	enableContinuousTranscription *api.OptionalBool) (transcriptionRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	interactionCreateTranscriptionRequest := &api.InteractionCreateTranscriptionRequest{
		Language:              language,
		EmbeddedGrammars:      embeddedGrammars,
		VadSettings:           vadSettings,
		AudioConsumeSettings:  audioConsumeSettings,
		NormalizationSettings: normalizationSettings,
		RecognitionSettings:   recognitionSettings,
	}

	if languageModelName != "" {
		interactionCreateTranscriptionRequest.LanguageModelName = &api.OptionalString{Value: languageModelName}
	}
	if acousticModelName != "" {
		interactionCreateTranscriptionRequest.AcousticModelName = &api.OptionalString{Value: acousticModelName}
	}
	if enablePostProcessing != "" {
		interactionCreateTranscriptionRequest.EnablePostprocessing = &api.OptionalString{Value: enablePostProcessing}
	}
	if enableContinuousTranscription != nil {
		interactionCreateTranscriptionRequest.ContinuousUtteranceTranscription = enableContinuousTranscription
	}

	transcriptionRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateTranscription{
					InteractionCreateTranscription: interactionCreateTranscriptionRequest,
				},
			},
		},
	}

	return transcriptionRequest
}

// getNormalizationRequest returns a normalization interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be auto-generated.
func getNormalizationRequest(
	correlationId string,
	language string,
	textToNormalize string,
	normalizationSettings *api.NormalizationSettings,
	generalInteractionSettings *api.GeneralInteractionSettings,
) (normalizationRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	normalizationRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateNormalizeText{
					InteractionCreateNormalizeText: &api.InteractionCreateNormalizeTextRequest{
						Language:                   language,
						Transcript:                 textToNormalize,
						NormalizationSettings:      normalizationSettings,
						GeneralInteractionSettings: generalInteractionSettings,
					},
				},
			},
		},
	}

	return normalizationRequest
}

// getAmdRequest returns an AMD interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getAmdRequest(
	correlationId string,
	amdSettings *api.AmdSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	vadSettings *api.VadSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (amdRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	amdRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateAmd{
					InteractionCreateAmd: &api.InteractionCreateAmdRequest{
						AmdSettings:                amdSettings,
						AudioConsumeSettings:       audioConsumeSettings,
						VadSettings:                vadSettings,
						GeneralInteractionSettings: generalInteractionSettings,
					},
				},
			},
		},
	}

	return amdRequest
}

// getCpaRequest returns an CPA interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getCpaRequest(
	correlationId string,
	cpaSettings *api.CpaSettings,
	audioConsumeSettings *api.AudioConsumeSettings,
	vadSettings *api.VadSettings,
	generalInteractionSettings *api.GeneralInteractionSettings) (cpaRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	cpaRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateCpa{
					InteractionCreateCpa: &api.InteractionCreateCpaRequest{
						CpaSettings:                cpaSettings,
						AudioConsumeSettings:       audioConsumeSettings,
						VadSettings:                vadSettings,
						GeneralInteractionSettings: generalInteractionSettings,
					},
				},
			},
		},
	}

	return cpaRequest
}

// getNluRequest returns an NLU interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be auto-generated.
func getNluRequest(
	correlationId string,
	language string,
	inputText string,
	nluSettings *api.NluSettings,
	generalInteractionSettings *api.GeneralInteractionSettings,
) (nluRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	nluRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateNlu{
					InteractionCreateNlu: &api.InteractionCreateNluRequest{
						Language:                   language,
						InputText:                  inputText,
						NluSettings:                nluSettings,
						GeneralInteractionSettings: generalInteractionSettings,
					},
				},
			},
		},
	}

	return nluRequest
}

// getInlineTtsRequest returns an inline TTS interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be auto-generated.
func getInlineTtsRequest(
	correlationId string,
	language string,
	textToSynthesize string,
	audioFormat *api.AudioFormat,
	synthesisTimeoutMs *api.OptionalInt32,
	inlineSynthesisSettings *api.TtsInlineSynthesisSettings,
	generalInteractionSettings *api.GeneralInteractionSettings,
) (ttsRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	requestBody := &api.InteractionCreateTtsRequest{
		Language:                   language,
		AudioFormat:                audioFormat,
		SynthesisTimeoutMs:         synthesisTimeoutMs,
		GeneralInteractionSettings: generalInteractionSettings,
		TtsRequest: &api.InteractionCreateTtsRequest_InlineRequest{
			InlineRequest: &api.InteractionCreateTtsRequest_InlineTtsRequest{
				Text:                       textToSynthesize,
				TtsInlineSynthesisSettings: inlineSynthesisSettings,
			},
		},
	}

	ttsRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateTts{
					InteractionCreateTts: requestBody,
				},
			},
		},
	}

	return ttsRequest
}

// getUrlTtsRequest returns a URL-based TTS interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be auto-generated.
func getUrlTtsRequest(
	correlationId string,
	language string,
	ssmlUrl string,
	audioFormat *api.AudioFormat,
	synthesisTimeoutMs *api.OptionalInt32,
	sslVerifyPeer *api.OptionalBool,
	generalInteractionSettings *api.GeneralInteractionSettings,
) (ttsRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	requestBody := &api.InteractionCreateTtsRequest{
		Language:                   language,
		AudioFormat:                audioFormat,
		SynthesisTimeoutMs:         synthesisTimeoutMs,
		GeneralInteractionSettings: generalInteractionSettings,
		TtsRequest: &api.InteractionCreateTtsRequest_SsmlRequest{
			SsmlRequest: &api.InteractionCreateTtsRequest_SsmlUrlRequest{
				SsmlUrl:       ssmlUrl,
				SslVerifyPeer: sslVerifyPeer,
			},
		},
	}

	ttsRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionCreateTts{
					InteractionCreateTts: requestBody,
				},
			},
		},
	}

	return ttsRequest
}

// getAudioPushRequest returns an audio push request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getAudioPushRequest(correlationId string, audioChunk []byte) (audioPushRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	audioPushRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_AudioRequest{
			AudioRequest: &api.AudioRequestMessage{
				AudioRequest: &api.AudioRequestMessage_AudioPush{
					AudioPush: &api.AudioPushRequest{
						AudioData: audioChunk,
					},
				},
			},
		},
	}

	return audioPushRequest
}

// getInteractionFinalizeRequest returns an interaction finalize request.
// The specified correlationId will be used if nonempty. Otherwise, one
// will be auto-generated.
func getInteractionFinalizeRequest(correlationId string, interactionId string) (finalizeRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	finalizeRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_InteractionRequest{
			InteractionRequest: &api.InteractionRequestMessage{
				InteractionRequest: &api.InteractionRequestMessage_InteractionFinalizeProcessing{
					InteractionFinalizeProcessing: &api.InteractionFinalizeProcessingRequest{
						InteractionId: interactionId,
					},
				},
			},
		},
	}

	return finalizeRequest
}

// getAudioPullRequest returns an audio pull request.
// The specified correlationId will be used if nonempty. Otherwise, one
// will be auto-generated.
func getAudioPullRequest(correlationId string, interactionId string, audioChannel int32, audioStartMs int32,
	audioLengthMs int32) (audioPullRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	audioPullRequestBody := &api.AudioPullRequest{
		AudioId:      interactionId,
		AudioChannel: nil,
		AudioStart:   nil,
		AudioLength:  nil,
	}

	if audioChannel != 0 {
		audioPullRequestBody.AudioChannel = &api.OptionalInt32{Value: audioChannel}
	}
	if audioStartMs != 0 {
		audioPullRequestBody.AudioStart = &api.OptionalInt32{Value: audioStartMs}
	}
	if audioLengthMs != 0 {
		audioPullRequestBody.AudioLength = &api.OptionalInt32{Value: audioLengthMs}
	}

	audioPullRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_AudioRequest{
			AudioRequest: &api.AudioRequestMessage{
				AudioRequest: &api.AudioRequestMessage_AudioPull{
					AudioPull: audioPullRequestBody,
				},
			},
		},
	}

	return audioPullRequest
}

// getSessionCloseRequest returns a session close request.
// The specified correlationId will be used if nonempty. Otherwise, one
// will be auto-generated.
func getSessionCloseRequest(correlationId string) (sessionCloseRequest *api.SessionRequest) {

	if correlationId == "" {
		// Create a new correlationId if one is not specified
		correlationId = uuid.NewString()
	}

	sessionCloseRequest = &api.SessionRequest{
		CorrelationId: &api.OptionalString{Value: correlationId},
		RequestType: &api.SessionRequest_SessionRequest{
			SessionRequest: &api.SessionRequestMessage{
				Name: &api.SessionRequestMessage_SessionClose{
					SessionClose: &api.SessionCloseRequest{},
				},
			},
		},
	}

	return sessionCloseRequest
}
