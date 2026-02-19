// Package session provides session and interaction lifecycle management for
// the LumenVox API.
//
// A [SessionObject] represents an active bidirectional gRPC stream with the
// API. It is created by calling [client.SdkClient.NewSession] and supports
// concurrent interaction management, audio streaming, and result consumption.
//
// # Interactions
//
// Each AI capability is represented by an interaction object with its own
// creation method on [SessionObject]:
//
//   - [SessionObject.NewAsr] — automatic speech recognition.
//   - [SessionObject.NewTranscription] — speech-to-text transcription.
//   - [SessionObject.NewInlineTts] / [SessionObject.NewUrlTts] — text-to-speech synthesis.
//   - [SessionObject.NewNlu] — natural language understanding (summarization, translation).
//   - [SessionObject.NewSentiment] — sentiment analysis.
//   - [SessionObject.NewDiarization] — speaker diarization.
//   - [SessionObject.NewLanguageId] — language identification.
//   - [SessionObject.NewAmd] — answering machine detection.
//   - [SessionObject.NewCpa] — call progress analysis.
//   - [SessionObject.NewNormalization] — text normalization and redaction.
//   - [SessionObject.NewGrammarParse] — grammar-based parsing.
//   - [SessionObject.NewNeuron] — intent detection.
//   - [SessionObject.NewInlineLoadSessionGrammar] / [SessionObject.NewUriLoadSessionGrammar] — session grammar loading.
//
// # Audio
//
// Use [SessionObject.AddAudio] to stream or batch audio data into the
// session. The SDK handles chunking and internal buffering automatically.
//
// # Results
//
// Interaction objects provide blocking Wait and Get methods for consuming
// results. For example, [AsrInteractionObject.WaitForFinalResults] blocks
// until the API returns final results or times out.
package session
