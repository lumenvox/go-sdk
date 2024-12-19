// API Protocol Buffer File
// This is the gRPC definition for the LumenVox API

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
//  protoc-gen-go v1.34.1
//  protoc        v5.27.3
// source: lumenvox/api/lumenvox.proto

package api

import (
    protoreflect "google.golang.org/protobuf/reflect/protoreflect"
    protoimpl "google.golang.org/protobuf/runtime/protoimpl"
    reflect "reflect"
)

const (
    // Verify that this generated code is sufficiently up-to-date.
    _ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
    // Verify that runtime/protoimpl is sufficiently up-to-date.
    _ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_lumenvox_api_lumenvox_proto protoreflect.FileDescriptor

var file_lumenvox_api_lumenvox_proto_rawDesc = []byte{
    0x0a, 0x1b, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6c,
    0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c, 0x6c,
    0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61, 0x70, 0x69, 0x1a, 0x19, 0x6c, 0x75, 0x6d,
    0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c,
    0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1a, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78,
    0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
    0x74, 0x6f, 0x32, 0x9f, 0x01, 0x0a, 0x08, 0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x12,
    0x4a, 0x0a, 0x07, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1c, 0x2e, 0x6c, 0x75, 0x6d,
    0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f,
    0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1d, 0x2e, 0x6c, 0x75, 0x6d, 0x65, 0x6e,
    0x76, 0x6f, 0x78, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x52,
    0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x28, 0x01, 0x30, 0x01, 0x12, 0x47, 0x0a, 0x06, 0x47,
    0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x12, 0x1b, 0x2e, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78,
    0x2e, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65,
    0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61, 0x70,
    0x69, 0x2e, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
    0x28, 0x01, 0x30, 0x01, 0x42, 0x8d, 0x01, 0x0a, 0x0c, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f,
    0x78, 0x2e, 0x61, 0x70, 0x69, 0x42, 0x0d, 0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x50,
    0x72, 0x6f, 0x74, 0x6f, 0x50, 0x00, 0x5a, 0x3d, 0x64, 0x65, 0x76, 0x2e, 0x61, 0x7a, 0x75, 0x72,
    0x65, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x2f, 0x6d,
    0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x2f, 0x5f, 0x67, 0x69, 0x74, 0x2f, 0x64, 0x65, 0x76, 0x70,
    0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x67, 0x69, 0x74, 0x2f, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f,
    0x78, 0x2f, 0x61, 0x70, 0x69, 0xa2, 0x02, 0x05, 0x43, 0x4c, 0x56, 0x4f, 0x50, 0xaa, 0x02, 0x0c,
    0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x2e, 0x41, 0x70, 0x69, 0xca, 0x02, 0x15, 0x4c,
    0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x5c, 0x41, 0x70, 0x69, 0x5c, 0x4c, 0x75, 0x6d, 0x65,
    0x6e, 0x76, 0x6f, 0x78, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_lumenvox_api_lumenvox_proto_goTypes = []interface{}{
    (*SessionRequest)(nil),  // 0: lumenvox.api.SessionRequest
    (*GlobalRequest)(nil),   // 1: lumenvox.api.GlobalRequest
    (*SessionResponse)(nil), // 2: lumenvox.api.SessionResponse
    (*GlobalResponse)(nil),  // 3: lumenvox.api.GlobalResponse
}
var file_lumenvox_api_lumenvox_proto_depIdxs = []int32{
    0, // 0: lumenvox.api.LumenVox.Session:input_type -> lumenvox.api.SessionRequest
    1, // 1: lumenvox.api.LumenVox.Global:input_type -> lumenvox.api.GlobalRequest
    2, // 2: lumenvox.api.LumenVox.Session:output_type -> lumenvox.api.SessionResponse
    3, // 3: lumenvox.api.LumenVox.Global:output_type -> lumenvox.api.GlobalResponse
    2, // [2:4] is the sub-list for method output_type
    0, // [0:2] is the sub-list for method input_type
    0, // [0:0] is the sub-list for extension type_name
    0, // [0:0] is the sub-list for extension extendee
    0, // [0:0] is the sub-list for field type_name
}

func init() { file_lumenvox_api_lumenvox_proto_init() }
func file_lumenvox_api_lumenvox_proto_init() {
    if File_lumenvox_api_lumenvox_proto != nil {
        return
    }
    file_lumenvox_api_global_proto_init()
    file_lumenvox_api_session_proto_init()
    type x struct{}
    out := protoimpl.TypeBuilder{
        File: protoimpl.DescBuilder{
            GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
            RawDescriptor: file_lumenvox_api_lumenvox_proto_rawDesc,
            NumEnums:      0,
            NumMessages:   0,
            NumExtensions: 0,
            NumServices:   1,
        },
        GoTypes:           file_lumenvox_api_lumenvox_proto_goTypes,
        DependencyIndexes: file_lumenvox_api_lumenvox_proto_depIdxs,
    }.Build()
    File_lumenvox_api_lumenvox_proto = out.File
    file_lumenvox_api_lumenvox_proto_rawDesc = nil
    file_lumenvox_api_lumenvox_proto_goTypes = nil
    file_lumenvox_api_lumenvox_proto_depIdxs = nil
}
