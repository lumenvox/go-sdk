package session

import (
    "github.com/lumenvox/go-sdk/lumenvox/api"
    "github.com/google/uuid"
)

// getSessionCreateRequest returns a create session request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getSessionCreateRequest(correlationId string, deploymentId string, operatorId string) *api.SessionRequest {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    res := &api.SessionRequest{
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

    return res
}

// getAudioFormatRequest returns an audio format request. The specified
// correlationId will be used if nonempty. Otherwise, one will be
// auto-generated.
func getAudioFormatRequest(correlationId string, audioConfig AudioConfig) (newRequest *api.SessionRequest) {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    newRequest = &api.SessionRequest{
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

    return newRequest
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
    vadSettings *api.VadSettings,
    audioConsumeSettings *api.AudioConsumeSettings,
    normalizationSettings *api.NormalizationSettings,
    recognitionSettings *api.RecognitionSettings,
    languageModelName string,
    acousticModelName string,
    enablePostProcessing string) (transcriptionRequest *api.SessionRequest) {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    interactionCreateTranscriptionRequest := &api.InteractionCreateTranscriptionRequest{
        Language: language,
        //EmbeddedGrammars:      inlineGrammarsRequest(),
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
) (transcriptionRequest *api.SessionRequest) {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    transcriptionRequest = &api.SessionRequest{
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

    return transcriptionRequest
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

// getSsmlTtsRequest returns an SSML TTS interaction request. The specified
// correlationId will be used if nonempty. Otherwise, one will be auto-generated.
func getSsmlTtsRequest(
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
func getAudioPushRequest(correlationId string, audioChunk []byte) (newAudioPushMessage *api.SessionRequest) {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    newAudioPushMessage = &api.SessionRequest{
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

    return newAudioPushMessage
}

// getInteractionFinalizeRequest returns an interaction finalize request.
// The specified correlationId will be used if nonempty. Otherwise, one
// will be auto-generated.
func getInteractionFinalizeRequest(correlationId string, interactionId string) (newRequest *api.SessionRequest) {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    newRequest = &api.SessionRequest{
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

    return newRequest
}

// getAudioPullRequest returns an audio pull request.
// The specified correlationId will be used if nonempty. Otherwise, one
// will be auto-generated.
func getAudioPullRequest(correlationId string, interactionId string, audioChannel int32, audioStartMs int32,
    audioLengthMs int32) (newRequest *api.SessionRequest) {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    audioPullRequest := &api.AudioPullRequest{
        AudioId:      interactionId,
        AudioChannel: nil,
        AudioStart:   nil,
        AudioLength:  nil,
    }

    if audioChannel != 0 {
        audioPullRequest.AudioChannel = &api.OptionalInt32{Value: audioChannel}
    }
    if audioStartMs != 0 {
        audioPullRequest.AudioStart = &api.OptionalInt32{Value: audioStartMs}
    }
    if audioLengthMs != 0 {
        audioPullRequest.AudioLength = &api.OptionalInt32{Value: audioLengthMs}
    }

    newRequest = &api.SessionRequest{
        CorrelationId: &api.OptionalString{Value: correlationId},
        RequestType: &api.SessionRequest_AudioRequest{
            AudioRequest: &api.AudioRequestMessage{
                AudioRequest: &api.AudioRequestMessage_AudioPull{
                    AudioPull: audioPullRequest,
                },
            },
        },
    }

    return newRequest
}

// getSessionCloseRequest returns a session close request.
// The specified correlationId will be used if nonempty. Otherwise, one
// will be auto-generated.
func getSessionCloseRequest(correlationId string) (newRequest *api.SessionRequest) {

    if correlationId == "" {
        // Create a new correlationId if one is not specified
        correlationId = uuid.NewString()
    }

    newRequest = &api.SessionRequest{
        CorrelationId: &api.OptionalString{Value: correlationId},
        RequestType: &api.SessionRequest_SessionRequest{
            SessionRequest: &api.SessionRequestMessage{
                Name: &api.SessionRequestMessage_SessionClose{
                    SessionClose: &api.SessionCloseRequest{},
                },
            },
        },
    }

    return newRequest
}
