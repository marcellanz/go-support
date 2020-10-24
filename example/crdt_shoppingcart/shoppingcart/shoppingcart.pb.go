// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.11.2
// source: shoppingcart.proto

package shoppingcart

import (
	_ "github.com/cloudstateio/go-support/cloudstate"
	empty "github.com/golang/protobuf/ptypes/empty"
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

type AddLineItem struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId    string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	ProductId string `protobuf:"bytes,2,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
	Name      string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Quantity  int32  `protobuf:"varint,4,opt,name=quantity,proto3" json:"quantity,omitempty"`
}

func (x *AddLineItem) Reset() {
	*x = AddLineItem{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shoppingcart_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddLineItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddLineItem) ProtoMessage() {}

func (x *AddLineItem) ProtoReflect() protoreflect.Message {
	mi := &file_shoppingcart_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddLineItem.ProtoReflect.Descriptor instead.
func (*AddLineItem) Descriptor() ([]byte, []int) {
	return file_shoppingcart_proto_rawDescGZIP(), []int{0}
}

func (x *AddLineItem) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *AddLineItem) GetProductId() string {
	if x != nil {
		return x.ProductId
	}
	return ""
}

func (x *AddLineItem) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *AddLineItem) GetQuantity() int32 {
	if x != nil {
		return x.Quantity
	}
	return 0
}

type RemoveLineItem struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId    string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	ProductId string `protobuf:"bytes,2,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
}

func (x *RemoveLineItem) Reset() {
	*x = RemoveLineItem{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shoppingcart_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RemoveLineItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveLineItem) ProtoMessage() {}

func (x *RemoveLineItem) ProtoReflect() protoreflect.Message {
	mi := &file_shoppingcart_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveLineItem.ProtoReflect.Descriptor instead.
func (*RemoveLineItem) Descriptor() ([]byte, []int) {
	return file_shoppingcart_proto_rawDescGZIP(), []int{1}
}

func (x *RemoveLineItem) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *RemoveLineItem) GetProductId() string {
	if x != nil {
		return x.ProductId
	}
	return ""
}

type GetShoppingCart struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
}

func (x *GetShoppingCart) Reset() {
	*x = GetShoppingCart{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shoppingcart_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetShoppingCart) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetShoppingCart) ProtoMessage() {}

func (x *GetShoppingCart) ProtoReflect() protoreflect.Message {
	mi := &file_shoppingcart_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetShoppingCart.ProtoReflect.Descriptor instead.
func (*GetShoppingCart) Descriptor() ([]byte, []int) {
	return file_shoppingcart_proto_rawDescGZIP(), []int{2}
}

func (x *GetShoppingCart) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

type LineItem struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProductId string `protobuf:"bytes,1,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
	Name      string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Quantity  int32  `protobuf:"varint,3,opt,name=quantity,proto3" json:"quantity,omitempty"`
}

func (x *LineItem) Reset() {
	*x = LineItem{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shoppingcart_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LineItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LineItem) ProtoMessage() {}

func (x *LineItem) ProtoReflect() protoreflect.Message {
	mi := &file_shoppingcart_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LineItem.ProtoReflect.Descriptor instead.
func (*LineItem) Descriptor() ([]byte, []int) {
	return file_shoppingcart_proto_rawDescGZIP(), []int{3}
}

func (x *LineItem) GetProductId() string {
	if x != nil {
		return x.ProductId
	}
	return ""
}

func (x *LineItem) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *LineItem) GetQuantity() int32 {
	if x != nil {
		return x.Quantity
	}
	return 0
}

type Cart struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Items []*LineItem `protobuf:"bytes,1,rep,name=items,proto3" json:"items,omitempty"`
}

func (x *Cart) Reset() {
	*x = Cart{}
	if protoimpl.UnsafeEnabled {
		mi := &file_shoppingcart_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Cart) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cart) ProtoMessage() {}

func (x *Cart) ProtoReflect() protoreflect.Message {
	mi := &file_shoppingcart_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cart.ProtoReflect.Descriptor instead.
func (*Cart) Descriptor() ([]byte, []int) {
	return file_shoppingcart_proto_rawDescGZIP(), []int{4}
}

func (x *Cart) GetItems() []*LineItem {
	if x != nil {
		return x.Items
	}
	return nil
}

var File_shoppingcart_proto protoreflect.FileDescriptor

var file_shoppingcart_proto_rawDesc = []byte{
	0x0a, 0x12, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x73, 0x68,
	0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x2f, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x5f, 0x6b, 0x65, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x7b, 0x0a, 0x0b, 0x41, 0x64, 0x64, 0x4c, 0x69, 0x6e, 0x65, 0x49,
	0x74, 0x65, 0x6d, 0x12, 0x1d, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x04, 0x90, 0xb5, 0x18, 0x01, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72,
	0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x5f, 0x69, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x49,
	0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x69, 0x74,
	0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x69, 0x74,
	0x79, 0x22, 0x4e, 0x0a, 0x0e, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4c, 0x69, 0x6e, 0x65, 0x49,
	0x74, 0x65, 0x6d, 0x12, 0x1d, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x04, 0x90, 0xb5, 0x18, 0x01, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72,
	0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x5f, 0x69, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x49,
	0x64, 0x22, 0x30, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x53, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x43, 0x61, 0x72, 0x74, 0x12, 0x1d, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x04, 0x90, 0xb5, 0x18, 0x01, 0x52, 0x06, 0x75, 0x73, 0x65,
	0x72, 0x49, 0x64, 0x22, 0x59, 0x0a, 0x08, 0x4c, 0x69, 0x6e, 0x65, 0x49, 0x74, 0x65, 0x6d, 0x12,
	0x1d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x49, 0x64, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x22, 0x3c,
	0x0a, 0x04, 0x43, 0x61, 0x72, 0x74, 0x12, 0x34, 0x0a, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e,
	0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e, 0x4c, 0x69, 0x6e,
	0x65, 0x49, 0x74, 0x65, 0x6d, 0x52, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x32, 0xc7, 0x02, 0x0a,
	0x13, 0x53, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x43, 0x61, 0x72, 0x74, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x44, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x49, 0x74, 0x65, 0x6d, 0x12,
	0x21, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69,
	0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e, 0x41, 0x64, 0x64, 0x4c, 0x69, 0x6e, 0x65, 0x49, 0x74,
	0x65, 0x6d, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x4a, 0x0a, 0x0a, 0x52, 0x65,
	0x6d, 0x6f, 0x76, 0x65, 0x49, 0x74, 0x65, 0x6d, 0x12, 0x24, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70,
	0x6c, 0x65, 0x2e, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e,
	0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x4c, 0x69, 0x6e, 0x65, 0x49, 0x74, 0x65, 0x6d, 0x1a, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x4c, 0x0a, 0x07, 0x47, 0x65, 0x74, 0x43, 0x61, 0x72,
	0x74, 0x12, 0x25, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x73, 0x68, 0x6f, 0x70,
	0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x68, 0x6f, 0x70,
	0x70, 0x69, 0x6e, 0x67, 0x43, 0x61, 0x72, 0x74, 0x1a, 0x1a, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70,
	0x6c, 0x65, 0x2e, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e,
	0x43, 0x61, 0x72, 0x74, 0x12, 0x50, 0x0a, 0x09, 0x57, 0x61, 0x74, 0x63, 0x68, 0x43, 0x61, 0x72,
	0x74, 0x12, 0x25, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x73, 0x68, 0x6f, 0x70,
	0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x68, 0x6f, 0x70,
	0x70, 0x69, 0x6e, 0x67, 0x43, 0x61, 0x72, 0x74, 0x1a, 0x1a, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70,
	0x6c, 0x65, 0x2e, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x2e,
	0x43, 0x61, 0x72, 0x74, 0x30, 0x01, 0x42, 0x65, 0x0a, 0x0b, 0x63, 0x6f, 0x6d, 0x2e, 0x65, 0x78,
	0x61, 0x6d, 0x70, 0x6c, 0x65, 0x5a, 0x56, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x73, 0x74, 0x61, 0x74, 0x65, 0x69, 0x6f, 0x2f, 0x67,
	0x6f, 0x2d, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2f, 0x63, 0x72, 0x64, 0x74, 0x5f, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63,
	0x61, 0x72, 0x74, 0x2f, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74,
	0x3b, 0x73, 0x68, 0x6f, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x63, 0x61, 0x72, 0x74, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_shoppingcart_proto_rawDescOnce sync.Once
	file_shoppingcart_proto_rawDescData = file_shoppingcart_proto_rawDesc
)

func file_shoppingcart_proto_rawDescGZIP() []byte {
	file_shoppingcart_proto_rawDescOnce.Do(func() {
		file_shoppingcart_proto_rawDescData = protoimpl.X.CompressGZIP(file_shoppingcart_proto_rawDescData)
	})
	return file_shoppingcart_proto_rawDescData
}

var file_shoppingcart_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_shoppingcart_proto_goTypes = []interface{}{
	(*AddLineItem)(nil),     // 0: example.shoppingcart.AddLineItem
	(*RemoveLineItem)(nil),  // 1: example.shoppingcart.RemoveLineItem
	(*GetShoppingCart)(nil), // 2: example.shoppingcart.GetShoppingCart
	(*LineItem)(nil),        // 3: example.shoppingcart.LineItem
	(*Cart)(nil),            // 4: example.shoppingcart.Cart
	(*empty.Empty)(nil),     // 5: google.protobuf.Empty
}
var file_shoppingcart_proto_depIdxs = []int32{
	3, // 0: example.shoppingcart.Cart.items:type_name -> example.shoppingcart.LineItem
	0, // 1: example.shoppingcart.ShoppingCartService.AddItem:input_type -> example.shoppingcart.AddLineItem
	1, // 2: example.shoppingcart.ShoppingCartService.RemoveItem:input_type -> example.shoppingcart.RemoveLineItem
	2, // 3: example.shoppingcart.ShoppingCartService.GetCart:input_type -> example.shoppingcart.GetShoppingCart
	2, // 4: example.shoppingcart.ShoppingCartService.WatchCart:input_type -> example.shoppingcart.GetShoppingCart
	5, // 5: example.shoppingcart.ShoppingCartService.AddItem:output_type -> google.protobuf.Empty
	5, // 6: example.shoppingcart.ShoppingCartService.RemoveItem:output_type -> google.protobuf.Empty
	4, // 7: example.shoppingcart.ShoppingCartService.GetCart:output_type -> example.shoppingcart.Cart
	4, // 8: example.shoppingcart.ShoppingCartService.WatchCart:output_type -> example.shoppingcart.Cart
	5, // [5:9] is the sub-list for method output_type
	1, // [1:5] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_shoppingcart_proto_init() }
func file_shoppingcart_proto_init() {
	if File_shoppingcart_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_shoppingcart_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddLineItem); i {
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
		file_shoppingcart_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RemoveLineItem); i {
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
		file_shoppingcart_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetShoppingCart); i {
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
		file_shoppingcart_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LineItem); i {
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
		file_shoppingcart_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Cart); i {
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
			RawDescriptor: file_shoppingcart_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_shoppingcart_proto_goTypes,
		DependencyIndexes: file_shoppingcart_proto_depIdxs,
		MessageInfos:      file_shoppingcart_proto_msgTypes,
	}.Build()
	File_shoppingcart_proto = out.File
	file_shoppingcart_proto_rawDesc = nil
	file_shoppingcart_proto_goTypes = nil
	file_shoppingcart_proto_depIdxs = nil
}
