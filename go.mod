module github.com/lumenvox/go-sdk

go 1.23.0

toolchain go1.23.2

require (
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
	github.com/google/uuid v1.6.0
	golang.org/x/oauth2 v0.27.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250224174004-546df14abb99
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.5
)

require cloud.google.com/go/compute/metadata v0.6.0 // indirect

require (
	github.com/go-audio/riff v1.0.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)
