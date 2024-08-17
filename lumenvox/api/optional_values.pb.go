// Protocol Buffer File
// This is the gRPC definition for optional value messages, and is loosely
// based on the google/protobuf/wrappers.proto wrapper definitions.
//
// These values are for use when a field is optional in LumenVox messages,
// which is internally used to distinguish between a field being specified
// with some default value or if the field is not being passed in the message.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
//  protoc-gen-go v1.34.1
//  protoc        v5.27.1
// source: lumenvox/api/optional_values.proto

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

// Wrapper message for optional `double`.
//
// The JSON representation for `OptionalDouble` is JSON number.
type OptionalDouble struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The double value.
    Value float64 `protobuf:"fixed64,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalDouble) Reset() {
    *x = OptionalDouble{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[0]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalDouble) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalDouble) ProtoMessage() {}

func (x *OptionalDouble) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[0]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalDouble.ProtoReflect.Descriptor instead.
func (*OptionalDouble) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{0}
}

func (x *OptionalDouble) GetValue() float64 {
    if x != nil {
        return x.Value
    }
    return 0
}

// Wrapper message for optional `float`.
//
// The JSON representation for `OptionalFloat` is JSON number.
type OptionalFloat struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The float value.
    Value float32 `protobuf:"fixed32,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalFloat) Reset() {
    *x = OptionalFloat{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[1]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalFloat) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalFloat) ProtoMessage() {}

func (x *OptionalFloat) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[1]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalFloat.ProtoReflect.Descriptor instead.
func (*OptionalFloat) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{1}
}

func (x *OptionalFloat) GetValue() float32 {
    if x != nil {
        return x.Value
    }
    return 0
}

// Wrapper message for optional `int64`.
//
// The JSON representation for `OptionalInt64` is JSON string.
type OptionalInt64 struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The int64 value.
    Value int64 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalInt64) Reset() {
    *x = OptionalInt64{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[2]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalInt64) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalInt64) ProtoMessage() {}

func (x *OptionalInt64) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[2]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalInt64.ProtoReflect.Descriptor instead.
func (*OptionalInt64) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{2}
}

func (x *OptionalInt64) GetValue() int64 {
    if x != nil {
        return x.Value
    }
    return 0
}

// Wrapper message for optional `uint64`.
//
// The JSON representation for `OptionalUInt64` is JSON string.
type OptionalUInt64 struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The uint64 value.
    Value uint64 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalUInt64) Reset() {
    *x = OptionalUInt64{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[3]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalUInt64) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalUInt64) ProtoMessage() {}

func (x *OptionalUInt64) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[3]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalUInt64.ProtoReflect.Descriptor instead.
func (*OptionalUInt64) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{3}
}

func (x *OptionalUInt64) GetValue() uint64 {
    if x != nil {
        return x.Value
    }
    return 0
}

// Wrapper message for optional `int32`.
//
// The JSON representation for `OptionalInt32` is JSON number.
type OptionalInt32 struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The int32 value.
    Value int32 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalInt32) Reset() {
    *x = OptionalInt32{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[4]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalInt32) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalInt32) ProtoMessage() {}

func (x *OptionalInt32) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[4]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalInt32.ProtoReflect.Descriptor instead.
func (*OptionalInt32) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{4}
}

func (x *OptionalInt32) GetValue() int32 {
    if x != nil {
        return x.Value
    }
    return 0
}

// Wrapper message for optional `uint32`.
//
// The JSON representation for `OptionalUInt32` is JSON number.
type OptionalUInt32 struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The uint32 value.
    Value uint32 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalUInt32) Reset() {
    *x = OptionalUInt32{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[5]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalUInt32) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalUInt32) ProtoMessage() {}

func (x *OptionalUInt32) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[5]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalUInt32.ProtoReflect.Descriptor instead.
func (*OptionalUInt32) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{5}
}

func (x *OptionalUInt32) GetValue() uint32 {
    if x != nil {
        return x.Value
    }
    return 0
}

// Wrapper message for optional `bool`.
//
// The JSON representation for `OptionalBool` is JSON `true` and `false`.
type OptionalBool struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The bool value.
    Value bool `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalBool) Reset() {
    *x = OptionalBool{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[6]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalBool) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalBool) ProtoMessage() {}

func (x *OptionalBool) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[6]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalBool.ProtoReflect.Descriptor instead.
func (*OptionalBool) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{6}
}

func (x *OptionalBool) GetValue() bool {
    if x != nil {
        return x.Value
    }
    return false
}

// Wrapper message for optional `string`.
//
// The JSON representation for `OptionalString` is JSON string.
type OptionalString struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The string value.
    Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalString) Reset() {
    *x = OptionalString{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[7]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalString) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalString) ProtoMessage() {}

func (x *OptionalString) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[7]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalString.ProtoReflect.Descriptor instead.
func (*OptionalString) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{7}
}

func (x *OptionalString) GetValue() string {
    if x != nil {
        return x.Value
    }
    return ""
}

// Wrapper message for optional `bytes`.
//
// The JSON representation for `OptionalBytes` is JSON string.
type OptionalBytes struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    // The bytes value.
    Value []byte `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *OptionalBytes) Reset() {
    *x = OptionalBytes{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lumenvox_api_optional_values_proto_msgTypes[8]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *OptionalBytes) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*OptionalBytes) ProtoMessage() {}

func (x *OptionalBytes) ProtoReflect() protoreflect.Message {
    mi := &file_lumenvox_api_optional_values_proto_msgTypes[8]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use OptionalBytes.ProtoReflect.Descriptor instead.
func (*OptionalBytes) Descriptor() ([]byte, []int) {
    return file_lumenvox_api_optional_values_proto_rawDescGZIP(), []int{8}
}

func (x *OptionalBytes) GetValue() []byte {
    if x != nil {
        return x.Value
    }
    return nil
}

var File_lumenvox_api_optional_values_proto protoreflect.FileDescriptor

var file_lumenvox_api_optional_values_proto_rawDesc = []byte{
    0x0a, 0x22, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6f,
    0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x2e, 0x70,
    0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2e, 0x61,
    0x70, 0x69, 0x22, 0x26, 0x0a, 0x0e, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x44, 0x6f,
    0x75, 0x62, 0x6c, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20,
    0x01, 0x28, 0x01, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x25, 0x0a, 0x0d, 0x4f, 0x70,
    0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x46, 0x6c, 0x6f, 0x61, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x76,
    0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x02, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
    0x65, 0x22, 0x25, 0x0a, 0x0d, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x49, 0x6e, 0x74,
    0x36, 0x34, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
    0x03, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x26, 0x0a, 0x0e, 0x4f, 0x70, 0x74, 0x69,
    0x6f, 0x6e, 0x61, 0x6c, 0x55, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
    0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
    0x22, 0x25, 0x0a, 0x0d, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x49, 0x6e, 0x74, 0x33,
    0x32, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05,
    0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x26, 0x0a, 0x0e, 0x4f, 0x70, 0x74, 0x69, 0x6f,
    0x6e, 0x61, 0x6c, 0x55, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
    0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22,
    0x24, 0x0a, 0x0c, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x42, 0x6f, 0x6f, 0x6c, 0x12,
    0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05,
    0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x26, 0x0a, 0x0e, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61,
    0x6c, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
    0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x25, 0x0a,
    0x0d, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x42, 0x79, 0x74, 0x65, 0x73, 0x12, 0x14,
    0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76,
    0x61, 0x6c, 0x75, 0x65, 0x42, 0xa8, 0x01, 0x0a, 0x0c, 0x6c, 0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f,
    0x78, 0x2e, 0x61, 0x70, 0x69, 0x42, 0x13, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x56,
    0x61, 0x6c, 0x75, 0x65, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x00, 0x5a, 0x3d, 0x64, 0x65,
    0x76, 0x2e, 0x61, 0x7a, 0x75, 0x72, 0x65, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x4c, 0x75, 0x6d, 0x65,
    0x6e, 0x56, 0x6f, 0x78, 0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x2f, 0x5f, 0x67, 0x69,
    0x74, 0x2f, 0x64, 0x65, 0x76, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x67, 0x69, 0x74, 0x2f, 0x6c,
    0x75, 0x6d, 0x65, 0x6e, 0x76, 0x6f, 0x78, 0x2f, 0x61, 0x70, 0x69, 0xa2, 0x02, 0x05, 0x43, 0x4c,
    0x56, 0x4f, 0x50, 0xaa, 0x02, 0x1b, 0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x2e, 0x41,
    0x70, 0x69, 0x2e, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65,
    0x73, 0xca, 0x02, 0x1b, 0x4c, 0x75, 0x6d, 0x65, 0x6e, 0x56, 0x6f, 0x78, 0x5c, 0x41, 0x70, 0x69,
    0x5c, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x62,
    0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
    file_lumenvox_api_optional_values_proto_rawDescOnce sync.Once
    file_lumenvox_api_optional_values_proto_rawDescData = file_lumenvox_api_optional_values_proto_rawDesc
)

func file_lumenvox_api_optional_values_proto_rawDescGZIP() []byte {
    file_lumenvox_api_optional_values_proto_rawDescOnce.Do(func() {
        file_lumenvox_api_optional_values_proto_rawDescData = protoimpl.X.CompressGZIP(file_lumenvox_api_optional_values_proto_rawDescData)
    })
    return file_lumenvox_api_optional_values_proto_rawDescData
}

var file_lumenvox_api_optional_values_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_lumenvox_api_optional_values_proto_goTypes = []interface{}{
    (*OptionalDouble)(nil), // 0: lumenvox.api.OptionalDouble
    (*OptionalFloat)(nil),  // 1: lumenvox.api.OptionalFloat
    (*OptionalInt64)(nil),  // 2: lumenvox.api.OptionalInt64
    (*OptionalUInt64)(nil), // 3: lumenvox.api.OptionalUInt64
    (*OptionalInt32)(nil),  // 4: lumenvox.api.OptionalInt32
    (*OptionalUInt32)(nil), // 5: lumenvox.api.OptionalUInt32
    (*OptionalBool)(nil),   // 6: lumenvox.api.OptionalBool
    (*OptionalString)(nil), // 7: lumenvox.api.OptionalString
    (*OptionalBytes)(nil),  // 8: lumenvox.api.OptionalBytes
}
var file_lumenvox_api_optional_values_proto_depIdxs = []int32{
    0, // [0:0] is the sub-list for method output_type
    0, // [0:0] is the sub-list for method input_type
    0, // [0:0] is the sub-list for extension type_name
    0, // [0:0] is the sub-list for extension extendee
    0, // [0:0] is the sub-list for field type_name
}

func init() { file_lumenvox_api_optional_values_proto_init() }
func file_lumenvox_api_optional_values_proto_init() {
    if File_lumenvox_api_optional_values_proto != nil {
        return
    }
    if !protoimpl.UnsafeEnabled {
        file_lumenvox_api_optional_values_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalDouble); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalFloat); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalInt64); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalUInt64); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalInt32); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalUInt32); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalBool); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalString); i {
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
        file_lumenvox_api_optional_values_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*OptionalBytes); i {
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
            RawDescriptor: file_lumenvox_api_optional_values_proto_rawDesc,
            NumEnums:      0,
            NumMessages:   9,
            NumExtensions: 0,
            NumServices:   0,
        },
        GoTypes:           file_lumenvox_api_optional_values_proto_goTypes,
        DependencyIndexes: file_lumenvox_api_optional_values_proto_depIdxs,
        MessageInfos:      file_lumenvox_api_optional_values_proto_msgTypes,
    }.Build()
    File_lumenvox_api_optional_values_proto = out.File
    file_lumenvox_api_optional_values_proto_rawDesc = nil
    file_lumenvox_api_optional_values_proto_goTypes = nil
    file_lumenvox_api_optional_values_proto_depIdxs = nil
}
