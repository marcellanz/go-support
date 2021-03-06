// Copyright 2019 Lightbend Inc.
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

// gRPC interface for Cloudstate Actions.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.11.2
// source: action.proto

package entity

import (
	protocol "github.com/cloudstateio/go-support/cloudstate/protocol"
	any "github.com/golang/protobuf/ptypes/any"
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

// An action command.
//
// For unary and streamed out calls, the service name, command name and payload will always be set.
//
// For streamed in and duplex streamed calls, the first command sent will just contain the service
// name and command name, but no payload. This will indicate that the action has been invoked.
// Subsequent commands on the stream will only have the payload set, the service name and command
// name will not be set.
type ActionCommand struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the service this action is on.
	ServiceName string `protobuf:"bytes,2,opt,name=service_name,json=serviceName,proto3" json:"service_name,omitempty"`
	// Command name
	Name string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	// The command payload.
	Payload *any.Any `protobuf:"bytes,4,opt,name=payload,proto3" json:"payload,omitempty"`
	// Metadata
	Metadata *protocol.Metadata `protobuf:"bytes,5,opt,name=metadata,proto3" json:"metadata,omitempty"`
}

func (x *ActionCommand) Reset() {
	*x = ActionCommand{}
	if protoimpl.UnsafeEnabled {
		mi := &file_action_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ActionCommand) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActionCommand) ProtoMessage() {}

func (x *ActionCommand) ProtoReflect() protoreflect.Message {
	mi := &file_action_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActionCommand.ProtoReflect.Descriptor instead.
func (*ActionCommand) Descriptor() ([]byte, []int) {
	return file_action_proto_rawDescGZIP(), []int{0}
}

func (x *ActionCommand) GetServiceName() string {
	if x != nil {
		return x.ServiceName
	}
	return ""
}

func (x *ActionCommand) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ActionCommand) GetPayload() *any.Any {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *ActionCommand) GetMetadata() *protocol.Metadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type ActionResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Response:
	//	*ActionResponse_Failure
	//	*ActionResponse_Reply
	//	*ActionResponse_Forward
	Response    isActionResponse_Response `protobuf_oneof:"response"`
	SideEffects []*protocol.SideEffect    `protobuf:"bytes,4,rep,name=side_effects,json=sideEffects,proto3" json:"side_effects,omitempty"`
}

func (x *ActionResponse) Reset() {
	*x = ActionResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_action_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ActionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActionResponse) ProtoMessage() {}

func (x *ActionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_action_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActionResponse.ProtoReflect.Descriptor instead.
func (*ActionResponse) Descriptor() ([]byte, []int) {
	return file_action_proto_rawDescGZIP(), []int{1}
}

func (m *ActionResponse) GetResponse() isActionResponse_Response {
	if m != nil {
		return m.Response
	}
	return nil
}

func (x *ActionResponse) GetFailure() *protocol.Failure {
	if x, ok := x.GetResponse().(*ActionResponse_Failure); ok {
		return x.Failure
	}
	return nil
}

func (x *ActionResponse) GetReply() *protocol.Reply {
	if x, ok := x.GetResponse().(*ActionResponse_Reply); ok {
		return x.Reply
	}
	return nil
}

func (x *ActionResponse) GetForward() *protocol.Forward {
	if x, ok := x.GetResponse().(*ActionResponse_Forward); ok {
		return x.Forward
	}
	return nil
}

func (x *ActionResponse) GetSideEffects() []*protocol.SideEffect {
	if x != nil {
		return x.SideEffects
	}
	return nil
}

type isActionResponse_Response interface {
	isActionResponse_Response()
}

type ActionResponse_Failure struct {
	Failure *protocol.Failure `protobuf:"bytes,1,opt,name=failure,proto3,oneof"`
}

type ActionResponse_Reply struct {
	Reply *protocol.Reply `protobuf:"bytes,2,opt,name=reply,proto3,oneof"`
}

type ActionResponse_Forward struct {
	Forward *protocol.Forward `protobuf:"bytes,3,opt,name=forward,proto3,oneof"`
}

func (*ActionResponse_Failure) isActionResponse_Response() {}

func (*ActionResponse_Reply) isActionResponse_Response() {}

func (*ActionResponse_Forward) isActionResponse_Response() {}

var File_action_proto protoreflect.FileDescriptor

var file_action_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x11,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x61, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x1a, 0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa8, 0x01, 0x0a, 0x0d, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2e,
	0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x30,
	0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x14, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x22, 0xe4, 0x01, 0x0a, 0x0e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x2f, 0x0a, 0x07, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2e, 0x46, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x48, 0x00, 0x52, 0x07, 0x66, 0x61, 0x69,
	0x6c, 0x75, 0x72, 0x65, 0x12, 0x29, 0x0a, 0x05, 0x72, 0x65, 0x70, 0x6c, 0x79, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x2e, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x48, 0x00, 0x52, 0x05, 0x72, 0x65, 0x70, 0x6c, 0x79, 0x12,
	0x2f, 0x0a, 0x07, 0x66, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x13, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x46, 0x6f,
	0x72, 0x77, 0x61, 0x72, 0x64, 0x48, 0x00, 0x52, 0x07, 0x66, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64,
	0x12, 0x39, 0x0a, 0x0c, 0x73, 0x69, 0x64, 0x65, 0x5f, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74, 0x73,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x2e, 0x53, 0x69, 0x64, 0x65, 0x45, 0x66, 0x66, 0x65, 0x63, 0x74, 0x52, 0x0b,
	0x73, 0x69, 0x64, 0x65, 0x45, 0x66, 0x66, 0x65, 0x63, 0x74, 0x73, 0x42, 0x0a, 0x0a, 0x08, 0x72,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0xfe, 0x02, 0x0a, 0x0e, 0x41, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x12, 0x54, 0x0a, 0x0b, 0x68, 0x61,
	0x6e, 0x64, 0x6c, 0x65, 0x55, 0x6e, 0x61, 0x72, 0x79, 0x12, 0x20, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x1a, 0x21, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e,
	0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00,
	0x12, 0x5b, 0x0a, 0x10, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x65, 0x64, 0x49, 0x6e, 0x12, 0x20, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2e, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43,
	0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x1a, 0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x2e, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x28, 0x01, 0x12, 0x5c, 0x0a,
	0x11, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x65, 0x64, 0x4f,
	0x75, 0x74, 0x12, 0x20, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e,
	0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6d,
	0x6d, 0x61, 0x6e, 0x64, 0x1a, 0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2e, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x30, 0x01, 0x12, 0x5b, 0x0a, 0x0e, 0x68,
	0x61, 0x6e, 0x64, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x65, 0x64, 0x12, 0x20, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x61, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x1a,
	0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x42, 0x55, 0x0a, 0x16, 0x69, 0x6f, 0x2e, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63,
	0x6f, 0x6c, 0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x69, 0x6f, 0x2f, 0x67, 0x6f, 0x2d, 0x73,
	0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2f, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x3b, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_action_proto_rawDescOnce sync.Once
	file_action_proto_rawDescData = file_action_proto_rawDesc
)

func file_action_proto_rawDescGZIP() []byte {
	file_action_proto_rawDescOnce.Do(func() {
		file_action_proto_rawDescData = protoimpl.X.CompressGZIP(file_action_proto_rawDescData)
	})
	return file_action_proto_rawDescData
}

var file_action_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_action_proto_goTypes = []interface{}{
	(*ActionCommand)(nil),       // 0: cloudstate.action.ActionCommand
	(*ActionResponse)(nil),      // 1: cloudstate.action.ActionResponse
	(*any.Any)(nil),             // 2: google.protobuf.Any
	(*protocol.Metadata)(nil),   // 3: cloudstate.Metadata
	(*protocol.Failure)(nil),    // 4: cloudstate.Failure
	(*protocol.Reply)(nil),      // 5: cloudstate.Reply
	(*protocol.Forward)(nil),    // 6: cloudstate.Forward
	(*protocol.SideEffect)(nil), // 7: cloudstate.SideEffect
}
var file_action_proto_depIdxs = []int32{
	2,  // 0: cloudstate.action.ActionCommand.payload:type_name -> google.protobuf.Any
	3,  // 1: cloudstate.action.ActionCommand.metadata:type_name -> cloudstate.Metadata
	4,  // 2: cloudstate.action.ActionResponse.failure:type_name -> cloudstate.Failure
	5,  // 3: cloudstate.action.ActionResponse.reply:type_name -> cloudstate.Reply
	6,  // 4: cloudstate.action.ActionResponse.forward:type_name -> cloudstate.Forward
	7,  // 5: cloudstate.action.ActionResponse.side_effects:type_name -> cloudstate.SideEffect
	0,  // 6: cloudstate.action.ActionProtocol.handleUnary:input_type -> cloudstate.action.ActionCommand
	0,  // 7: cloudstate.action.ActionProtocol.handleStreamedIn:input_type -> cloudstate.action.ActionCommand
	0,  // 8: cloudstate.action.ActionProtocol.handleStreamedOut:input_type -> cloudstate.action.ActionCommand
	0,  // 9: cloudstate.action.ActionProtocol.handleStreamed:input_type -> cloudstate.action.ActionCommand
	1,  // 10: cloudstate.action.ActionProtocol.handleUnary:output_type -> cloudstate.action.ActionResponse
	1,  // 11: cloudstate.action.ActionProtocol.handleStreamedIn:output_type -> cloudstate.action.ActionResponse
	1,  // 12: cloudstate.action.ActionProtocol.handleStreamedOut:output_type -> cloudstate.action.ActionResponse
	1,  // 13: cloudstate.action.ActionProtocol.handleStreamed:output_type -> cloudstate.action.ActionResponse
	10, // [10:14] is the sub-list for method output_type
	6,  // [6:10] is the sub-list for method input_type
	6,  // [6:6] is the sub-list for extension type_name
	6,  // [6:6] is the sub-list for extension extendee
	0,  // [0:6] is the sub-list for field type_name
}

func init() { file_action_proto_init() }
func file_action_proto_init() {
	if File_action_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_action_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ActionCommand); i {
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
		file_action_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ActionResponse); i {
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
	file_action_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*ActionResponse_Failure)(nil),
		(*ActionResponse_Reply)(nil),
		(*ActionResponse_Forward)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_action_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_action_proto_goTypes,
		DependencyIndexes: file_action_proto_depIdxs,
		MessageInfos:      file_action_proto_msgTypes,
	}.Build()
	File_action_proto = out.File
	file_action_proto_rawDesc = nil
	file_action_proto_goTypes = nil
	file_action_proto_depIdxs = nil
}
