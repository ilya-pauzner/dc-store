// Copyright 2015 gRPC authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.23.0
// 	protoc        v3.11.4
// source: validator.proto

package validator

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type ValidateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	Write bool   `protobuf:"varint,2,opt,name=write,proto3" json:"write,omitempty"`
}

func (x *ValidateRequest) Reset() {
	*x = ValidateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_validator_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidateRequest) ProtoMessage() {}

func (x *ValidateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_validator_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidateRequest.ProtoReflect.Descriptor instead.
func (*ValidateRequest) Descriptor() ([]byte, []int) {
	return file_validator_proto_rawDescGZIP(), []int{0}
}

func (x *ValidateRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *ValidateRequest) GetWrite() bool {
	if x != nil {
		return x.Write
	}
	return false
}

type ValidateReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

func (x *ValidateReply) Reset() {
	*x = ValidateReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_validator_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidateReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidateReply) ProtoMessage() {}

func (x *ValidateReply) ProtoReflect() protoreflect.Message {
	mi := &file_validator_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidateReply.ProtoReflect.Descriptor instead.
func (*ValidateReply) Descriptor() ([]byte, []int) {
	return file_validator_proto_rawDescGZIP(), []int{1}
}

func (x *ValidateReply) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

var File_validator_proto protoreflect.FileDescriptor

var file_validator_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x09, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x22, 0x3d, 0x0a, 0x0f,
	0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x77, 0x72, 0x69, 0x74, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x77, 0x72, 0x69, 0x74, 0x65, 0x22, 0x29, 0x0a, 0x0d, 0x56,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x18, 0x0a, 0x07,
	0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73,
	0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x32, 0x54, 0x0a, 0x09, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61,
	0x74, 0x6f, 0x72, 0x12, 0x47, 0x0a, 0x0d, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x54,
	0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x1a, 0x2e, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72,
	0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x18, 0x2e, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x2e, 0x56, 0x61, 0x6c,
	0x69, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x42, 0x2c, 0x5a, 0x2a,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x69, 0x6c, 0x79, 0x61, 0x2d,
	0x70, 0x61, 0x75, 0x7a, 0x6e, 0x65, 0x72, 0x2f, 0x64, 0x63, 0x2d, 0x73, 0x74, 0x6f, 0x72, 0x65,
	0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_validator_proto_rawDescOnce sync.Once
	file_validator_proto_rawDescData = file_validator_proto_rawDesc
)

func file_validator_proto_rawDescGZIP() []byte {
	file_validator_proto_rawDescOnce.Do(func() {
		file_validator_proto_rawDescData = protoimpl.X.CompressGZIP(file_validator_proto_rawDescData)
	})
	return file_validator_proto_rawDescData
}

var file_validator_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_validator_proto_goTypes = []interface{}{
	(*ValidateRequest)(nil), // 0: validator.ValidateRequest
	(*ValidateReply)(nil),   // 1: validator.ValidateReply
}
var file_validator_proto_depIdxs = []int32{
	0, // 0: validator.Validator.ValidateToken:input_type -> validator.ValidateRequest
	1, // 1: validator.Validator.ValidateToken:output_type -> validator.ValidateReply
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_validator_proto_init() }
func file_validator_proto_init() {
	if File_validator_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_validator_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ValidateRequest); i {
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
		file_validator_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ValidateReply); i {
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
			RawDescriptor: file_validator_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_validator_proto_goTypes,
		DependencyIndexes: file_validator_proto_depIdxs,
		MessageInfos:      file_validator_proto_msgTypes,
	}.Build()
	File_validator_proto = out.File
	file_validator_proto_rawDesc = nil
	file_validator_proto_goTypes = nil
	file_validator_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// ValidatorClient is the client API for Validator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ValidatorClient interface {
	ValidateToken(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateReply, error)
}

type validatorClient struct {
	cc grpc.ClientConnInterface
}

func NewValidatorClient(cc grpc.ClientConnInterface) ValidatorClient {
	return &validatorClient{cc}
}

func (c *validatorClient) ValidateToken(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateReply, error) {
	out := new(ValidateReply)
	err := c.cc.Invoke(ctx, "/validator.Validator/ValidateToken", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ValidatorServer is the server API for Validator service.
type ValidatorServer interface {
	ValidateToken(context.Context, *ValidateRequest) (*ValidateReply, error)
}

// UnimplementedValidatorServer can be embedded to have forward compatible implementations.
type UnimplementedValidatorServer struct {
}

func (*UnimplementedValidatorServer) ValidateToken(context.Context, *ValidateRequest) (*ValidateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ValidateToken not implemented")
}

func RegisterValidatorServer(s *grpc.Server, srv ValidatorServer) {
	s.RegisterService(&_Validator_serviceDesc, srv)
}

func _Validator_ValidateToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ValidatorServer).ValidateToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/validator.Validator/ValidateToken",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ValidatorServer).ValidateToken(ctx, req.(*ValidateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Validator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "validator.Validator",
	HandlerType: (*ValidatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ValidateToken",
			Handler:    _Validator_ValidateToken_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "validator.proto",
}
