module tts_inline_with_partial_streaming

go 1.27.2

require (
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
	github.com/lumenvox/go-sdk/v2 v2.0.0
	github.com/lumenvox/protos-go v1.80.1
)

require (
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261012220534-4341edf3f941 // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/ini.v1 v1.67.3 // indirect
)

// Development: Map GitHub SDK to local directory
replace github.com/lumenvox/go-sdk/v2 => ../..
