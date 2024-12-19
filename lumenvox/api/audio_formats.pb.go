// Protocol Buffer File
// This is the gRPC definition for AudioFormat messages

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
//  protoc-gen-go v1.34.1
//  protoc        v5.27.3
// source: lumenvox/api/audio_formats.proto

package api

import (
    protoreflect "google.golang.org/protobuf/reflect/protoreflect"
    protoimpl "google.golang.org/protobuf/runtime/protoimpl"
    reflect "reflect"
    sync "sync"
)

const (
    // Verify that this generated code is sufficiently up-to-date.
    _ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
    // Verify that runtime/protoimpl is sufficiently up-to-date.
    _ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Specification for the audio format
//
// Not all standard formats are supported in all cases. Different operations
// may natively handle a subset of the total audio formats.
type AudioFormat_StandardAudioFormat int32

const (
    AudioFormat_STANDARD_AUDIO_FORMAT_UNSPECIFIED AudioFormat_StandardAudioFormat = 0
    // Uncompressed 16-bit signed little-endian samples (Linear PCM).
    AudioFormat_STANDARD_AUDIO_FORMAT_LINEAR16 AudioFormat_StandardAudioFormat = 1
    // 8-bit audio samples using G.711 PCMU/mu-law.
    AudioFormat_STANDARD_AUDIO_FORMAT_ULAW AudioFormat_StandardAudioFormat = 2
    // 8-bit audio samples using G.711 PCMA/a-law.
    AudioFormat_STANDARD_AUDIO_FORMAT_ALAW AudioFormat_StandardAudioFormat = 3
    // WAV formatted audio
    AudioFormat_STANDARD_AUDIO_FORMAT_WAV AudioFormat_StandardAudioFormat = 4
    // FLAC formatted audio
    AudioFormat_STANDARD_AUDIO_FORMAT_FLAC AudioFormat_StandardAudioFormat = 5
    // MP3 formatted audio
    AudioFormat_STANDARD_AUDIO_FORMAT_MP3 AudioFormat_StandardAudioFormat = 6
    // OPUS formatted audio
    AudioFormat_STANDARD_AUDIO_FORMAT_OPUS AudioFormat_StandardAudioFormat = 7
    // M4A formatted audio
    AudioFormat_STANDARD_AUDIO_FORMAT_M4A AudioFormat_StandardAudioFormat = 8
    // Audio packed into MP4 container
    AudioFormat_STANDARD_AUDIO_FORMAT_MP4 AudioFormat_StandardAudioFormat = 9
    // Explicitly indicate that no audio resource should be allocated
    AudioFormat_STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE AudioFormat_StandardAudioFormat = 100
)

// Enum value maps for AudioFormat_StandardAudioFormat.
var (
    AudioFormat_StandardAudioFormat_name = map[int32]string{
        0:   "STANDARD_AUDIO_FORMAT_UNSPECIFIED",
        1:   "STANDARD_AUDIO_FORMAT_LINEAR16",
        2:   "STANDARD_AUDIO_FORMAT_ULAW",
        3:   "STANDARD_AUDIO_FORMAT_ALAW",
        4:   "STANDARD_AUDIO_FORMAT_WAV",
        5:   "STANDARD_AUDIO_FORMAT_FLAC",
        6:   "STANDARD_AUDIO_FORMAT_MP3",
        7:   "STANDARD_AUDIO_FORMAT_OPUS",
        8:   "STANDARD_AUDIO_FORMAT_M4A",
        9:   "STANDARD_AUDIO_FORMAT_MP4",
        100: "STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE",
    }
    AudioFormat_StandardAudioFormat_value = map[string]int32{
        "STANDARD_AUDIO_FORMAT_UNSPECIFIED":       0,
        "STANDARD_AUDIO_FORMAT_LINEAR16":          1,
        "STANDARD_AUDIO_FORMAT_ULAW":              2,
        "STANDARD_AUDIO_FORMAT_ALAW":              3,
        "STANDARD_AUDIO_FORMAT_WAV":               4,
        "STANDARD_AUDIO_FORMAT_FLAC":              5,
        "STANDARD_AUDIO_FORMAT_MP3":               6,
        "STANDARD_AUDIO_FORMAT_OPUS":              7,
        "STANDARD_AUDIO_FORMAT_M4A":               8,
        "STANDARD_AUDIO_FORMAT_MP4":               9,
        "STANDARD_AUDIO_FORMAT_NO_AUDIO_RESOURCE": 100,
    }
)

func (x AudioFormat_StandardAudioFormat) Enum() *AudioFormat_StandardAudioFormat {
    p := new(AudioFormat_StandardAudioFormat)
    *p = x
    return p
}

func (x AudioFormat_StandardAudioFormat) String() string {
    return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AudioFormat_StandardAudioFormat) Descriptor() protoreflect.EnumDescriptor {
    return file_lumenvox_api_audio_formats_proto_enumTypes[0].Descriptor()
}

func (AudioFormat_StandardAudioFormat) Type() protoreflect.EnumType {
    return &file_lumenvox_api_audio_formats_proto_enumTypes[0]
}

func (x AudioFormat_StandardAudioFormat) Number() protoreflect.EnumNumber {
    return protoreflect.EnumNumber(x)
}

// Deprecated: Use AudioFormat_StandardAudioFormat.Descriptor instead.
func (AudioFormat_StandardAudioFormat) EnumDescriptor() ([]byte, []int) {
    return file_lumenvox_api_audio_formats_proto_rawDescGZIP(), []int{0, 0}
}

type AudioFormat struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // Standard audio format
    StandardAudioFormat AudioFormat_StandardAudioFormat `protobuf:"varint,1,opt,name=standard_audio_format,json=standardAudioFormat,proto3,enum=lumenvox.api.AudioFormat_StandardAudioFormat" json:"standard_audio_format,omitempty"`
    // Sample rate in Hertz of the audio data
    // This field is mandatory for RAW PCM audio format. It's optional for the other formats.
    // For audio formats with headers, this value will be ignored, and instead the value from the file header will be used.
    // Default: 8000 (8 KHz)
    SampleRateHertz *OptionalInt32 `protobuf:"bytes,2,opt,name=sample_rate_hertz,json=sampleRateHertz,proto3" json:"sample_rate_hertz,omitempty"`
}

func (x *AudioFormat) Reset() {
    *x = AudioFormat{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_audio_formats_proto_msgTypes[0]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *AudioFormat) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*AudioFormat) ProtoMessage() {}

func (x *AudioFormat) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_audio_formats_proto_msgTypes[0]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use AudioFormat.ProtoReflect.Descriptor instead.
func (*AudioFormat) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_audio_formats_proto_rawDescGZIP(), []int{0}
}

func (x *AudioFormat) GetStandardAudioFormat() AudioFormat_StandardAudioFormat {
    if x != nil {
        return x.StandardAudioFormat
    }
    return AudioFormat_STANDARD_AUDIO_FORMAT_UNSPECIFIED
}

func (x *AudioFormat) GetSampleRateHertz() *OptionalInt32 {
    if x != nil {
        return x.SampleRateHertz
    }
    return nil
}

var File_lumenvox_api_audio_formats_proto protoreflect.FileDescriptor

var file_lumenvox_api_audio_formats_proto_rawDesc = []byte{
    0x0a, 0x20, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61,
    0x75, 0x64, 0x69, 0x6f, 0x5f, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f,
    0x74, 0x6f, 0x12, 0x0c, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61, 0x70, 0x69,
    0x1a, 0x22, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6f,
    0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x2e, 0x70,
    0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc5, 0x04, 0x0a, 0x0b, 0x41, 0x75, 0x64, 0x69, 0x6f, 0x46, 0x6f,
    0x72, 0x6d, 0x61, 0x74, 0x12, 0x61, 0x0a, 0x15, 0x73, 0x74, 0x61, 0x6e, 0x64, 0x61, 0x72, 0x64,
    0x5f, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x5f, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18, 0x01, 0x20,
    0x01, 0x28, 0x0e, 0x32, 0x2d, 0x2e, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61,
    0x70, 0x69, 0x2e, 0x41, 0x75, 0x64, 0x69, 0x6f, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x2e, 0x53,
    0x74, 0x61, 0x6e, 0x64, 0x61, 0x72, 0x64, 0x41, 0x75, 0x64, 0x69, 0x6f, 0x46, 0x6f, 0x72, 0x6d,
    0x61, 0x74, 0x52, 0x13, 0x73, 0x74, 0x61, 0x6e, 0x64, 0x61, 0x72, 0x64, 0x41, 0x75, 0x64, 0x69,
    0x6f, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x12, 0x47, 0x0a, 0x11, 0x73, 0x61, 0x6d, 0x70, 0x6c,
    0x65, 0x5f, 0x72, 0x61, 0x74, 0x65, 0x5f, 0x68, 0x65, 0x72, 0x74, 0x7a, 0x18, 0x02, 0x20, 0x01,
    0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61, 0x70,
    0x69, 0x2e, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x52,
    0x0f, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x52, 0x61, 0x74, 0x65, 0x48, 0x65, 0x72, 0x74, 0x7a,
    0x22, 0x89, 0x03, 0x0a, 0x13, 0x53, 0x74, 0x61, 0x6e, 0x64, 0x61, 0x72, 0x64, 0x41, 0x75, 0x64,
    0x69, 0x6f, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x12, 0x25, 0x0a, 0x21, 0x53, 0x54, 0x41, 0x4e,
    0x44, 0x41, 0x52, 0x44, 0x5f, 0x41, 0x55, 0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41,
    0x54, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12,
    0x22, 0x0a, 0x1e, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f, 0x41, 0x55, 0x44, 0x49,
    0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x4c, 0x49, 0x4e, 0x45, 0x41, 0x52, 0x31,
    0x36, 0x10, 0x01, 0x12, 0x1e, 0x0a, 0x1a, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f,
    0x41, 0x55, 0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x55, 0x4c, 0x41,
    0x57, 0x10, 0x02, 0x12, 0x1e, 0x0a, 0x1a, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f,
    0x41, 0x55, 0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x41, 0x4c, 0x41,
    0x57, 0x10, 0x03, 0x12, 0x1d, 0x0a, 0x19, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f,
    0x41, 0x55, 0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x57, 0x41, 0x56,
    0x10, 0x04, 0x12, 0x1e, 0x0a, 0x1a, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f, 0x41,
    0x55, 0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x46, 0x4c, 0x41, 0x43,
    0x10, 0x05, 0x12, 0x1d, 0x0a, 0x19, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f, 0x41,
    0x55, 0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x4d, 0x50, 0x33, 0x10,
    0x06, 0x12, 0x1e, 0x0a, 0x1a, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f, 0x41, 0x55,
    0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x4f, 0x50, 0x55, 0x53, 0x10,
    0x07, 0x12, 0x1d, 0x0a, 0x19, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f, 0x41, 0x55,
    0x44, 0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x4d, 0x34, 0x41, 0x10, 0x08,
    0x12, 0x1d, 0x0a, 0x19, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f, 0x41, 0x55, 0x44,
    0x49, 0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x4d, 0x50, 0x34, 0x10, 0x09, 0x12,
    0x2b, 0x0a, 0x27, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x5f, 0x41, 0x55, 0x44, 0x49,
    0x4f, 0x5f, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x4e, 0x4f, 0x5f, 0x41, 0x55, 0x44, 0x49,
    0x4f, 0x5f, 0x52, 0x45, 0x53, 0x4f, 0x55, 0x52, 0x43, 0x45, 0x10, 0x64, 0x42, 0xa2, 0x01, 0x0a,
    0x0c, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61, 0x70, 0x69, 0x42, 0x11, 0x41,
    0x75, 0x64, 0x69, 0x6f, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f,
    0x50, 0x00, 0x5a, 0x3d, 0x64, 0x65, 0x76, 0x2e, 0x61, 0x7a, 0x75, 0x72, 0x65, 0x2e, 0x63, 0x6f,
    0x6d, 0x2f, 0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c,
    0x65, 0x73, 0x2f, 0x5f, 0x67, 0x69, 0x74, 0x2f, 0x64, 0x65, 0x76, 0x70, 0x72, 0x6f, 0x74, 0x6f,
    0x2e, 0x67, 0x69, 0x74, 0x2f, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2f, 0x61, 0x70,
    0x69, 0xa2, 0x02, 0x05, 0x43, 0x4c, 0x56, 0x4f, 0x50, 0xaa, 0x02, 0x19, 0x4c, 0x75, 0x6d, 0x65,
    0x6e, 0x56, 0x6f, 0x78, 0x2e, 0x41, 0x70, 0x69, 0x2e, 0x41, 0x75, 0x64, 0x69, 0x6f, 0x46, 0x6f,
    0x72, 0x6d, 0x61, 0x74, 0x73, 0xca, 0x02, 0x19, 0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78,
    0x5c, 0x41, 0x70, 0x69, 0x5c, 0x41, 0x75, 0x64, 0x69, 0x6f, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74,
    0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
    file_lumenvox_api_audio_formats_proto_rawDescOnce sync.Once
    file_lumenvox_api_audio_formats_proto_rawDescData = file_lumenvox_api_audio_formats_proto_rawDesc
)

func file_lumenvox_api_audio_formats_proto_rawDescGZIP() []byte {
    file_lumenvox_api_audio_formats_proto_rawDescOnce.Do(func() {
        file_lumenvox_api_audio_formats_proto_rawDescData = protoimpl.X.CompressGZIP(file_lumenvox_api_audio_formats_proto_rawDescData)
    })
    return file_lumenvox_api_audio_formats_proto_rawDescData
}

var file_lumenvox_api_audio_formats_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_lumenvox_api_audio_formats_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_lumenvox_api_audio_formats_proto_goTypes = []interface{}{
    (AudioFormat_StandardAudioFormat)(0), // 0: lumenvox.api.AudioFormat.StandardAudioFormat
    (*AudioFormat)(nil),                  // 1: lumenvox.api.AudioFormat
    (*OptionalInt32)(nil),                // 2: lumenvox.api.OptionalInt32
}
var file_lumenvox_api_audio_formats_proto_depIdxs = []int32{
    0, // 0: lumenvox.api.AudioFormat.standard_audio_format:type_name -> lumenvox.api.AudioFormat.StandardAudioFormat
    2, // 1: lumenvox.api.AudioFormat.sample_rate_hertz:type_name -> lumenvox.api.OptionalInt32
    2, // [2:2] is the sub-list for method output_type
    2, // [2:2] is the sub-list for method input_type
    2, // [2:2] is the sub-list for extension type_name
    2, // [2:2] is the sub-list for extension extendee
    0, // [0:2] is the sub-list for field type_name
}

func init() { file_lumenvox_api_audio_formats_proto_init() }
func file_lumenvox_api_audio_formats_proto_init() {
    if File_lumenvox_api_audio_formats_proto != nil {
        return
    }
    file_lumenvox_api_optional_values_proto_init()
    if !protoimpl.UnsafeEnabled {
        file_lumenvox_api_audio_formats_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*AudioFormat); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
    }
    type x struct{}
    out := protoimpl.TypeBuilder{
        File: protoimpl.DescBuilder{
            GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
            RawDescriptor: file_lumenvox_api_audio_formats_proto_rawDesc,
            NumEnums:      1,
            NumMessages:   1,
            NumExtensions: 0,
            NumServices:   0,
        },
        GoTypes:           file_lumenvox_api_audio_formats_proto_goTypes,
        DependencyIndexes: file_lumenvox_api_audio_formats_proto_depIdxs,
        EnumInfos:         file_lumenvox_api_audio_formats_proto_enumTypes,
        MessageInfos:      file_lumenvox_api_audio_formats_proto_msgTypes,
    }.Build()
    File_lumenvox_api_audio_formats_proto = out.File
    file_lumenvox_api_audio_formats_proto_rawDesc = nil
    file_lumenvox_api_audio_formats_proto_goTypes = nil
    file_lumenvox_api_audio_formats_proto_depIdxs = nil
}
