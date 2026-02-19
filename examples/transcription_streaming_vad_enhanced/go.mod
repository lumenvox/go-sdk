module transcription_streaming_vad_enhanced

go 1.25.7

require (
	github.com/lumenvox/go-sdk v0.0.0
	github.com/lumenvox/protos-go v1.70.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260217215200-42d3e9bedb6d // indirect
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/ini.v1 v1.67.1 // indirect
)

// Development: Map GitHub SDK to local directory
replace github.com/lumenvox/go-sdk => ../..
