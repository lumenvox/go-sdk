package session

type GrammarObject struct {
	LoadedSuccessfully      bool
	LoadMessage             string
	IsBuffer                bool
	Language                string
	GrammarContent          string
	GrammarLabel            string
	IsVoiceGrammar          bool
	IsDTMFGrammar           bool
	IsTranscriptionGrammar  bool
	CompressInterpretations bool
}
