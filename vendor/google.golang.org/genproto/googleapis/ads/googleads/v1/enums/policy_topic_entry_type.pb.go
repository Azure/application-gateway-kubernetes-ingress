// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v1/enums/policy_topic_entry_type.proto

package enums // import "google.golang.org/genproto/googleapis/ads/googleads/v1/enums"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "google.golang.org/genproto/googleapis/api/annotations"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// The possible policy topic entry types.
type PolicyTopicEntryTypeEnum_PolicyTopicEntryType int32

const (
	// No value has been specified.
	PolicyTopicEntryTypeEnum_UNSPECIFIED PolicyTopicEntryTypeEnum_PolicyTopicEntryType = 0
	// The received value is not known in this version.
	//
	// This is a response-only value.
	PolicyTopicEntryTypeEnum_UNKNOWN PolicyTopicEntryTypeEnum_PolicyTopicEntryType = 1
	// The resource will not be served.
	PolicyTopicEntryTypeEnum_PROHIBITED PolicyTopicEntryTypeEnum_PolicyTopicEntryType = 2
	// The resource will not be served under some circumstances.
	PolicyTopicEntryTypeEnum_LIMITED PolicyTopicEntryTypeEnum_PolicyTopicEntryType = 4
	// May be of interest, but does not limit how the resource is served.
	PolicyTopicEntryTypeEnum_DESCRIPTIVE PolicyTopicEntryTypeEnum_PolicyTopicEntryType = 5
	// Could increase coverage beyond normal.
	PolicyTopicEntryTypeEnum_BROADENING PolicyTopicEntryTypeEnum_PolicyTopicEntryType = 6
	// Constrained for all targeted countries, but may serve in other countries
	// through area of interest.
	PolicyTopicEntryTypeEnum_AREA_OF_INTEREST_ONLY PolicyTopicEntryTypeEnum_PolicyTopicEntryType = 7
)

var PolicyTopicEntryTypeEnum_PolicyTopicEntryType_name = map[int32]string{
	0: "UNSPECIFIED",
	1: "UNKNOWN",
	2: "PROHIBITED",
	4: "LIMITED",
	5: "DESCRIPTIVE",
	6: "BROADENING",
	7: "AREA_OF_INTEREST_ONLY",
}
var PolicyTopicEntryTypeEnum_PolicyTopicEntryType_value = map[string]int32{
	"UNSPECIFIED":           0,
	"UNKNOWN":               1,
	"PROHIBITED":            2,
	"LIMITED":               4,
	"DESCRIPTIVE":           5,
	"BROADENING":            6,
	"AREA_OF_INTEREST_ONLY": 7,
}

func (x PolicyTopicEntryTypeEnum_PolicyTopicEntryType) String() string {
	return proto.EnumName(PolicyTopicEntryTypeEnum_PolicyTopicEntryType_name, int32(x))
}
func (PolicyTopicEntryTypeEnum_PolicyTopicEntryType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_policy_topic_entry_type_f2b35797c6ea84c5, []int{0, 0}
}

// Container for enum describing possible policy topic entry types.
type PolicyTopicEntryTypeEnum struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PolicyTopicEntryTypeEnum) Reset()         { *m = PolicyTopicEntryTypeEnum{} }
func (m *PolicyTopicEntryTypeEnum) String() string { return proto.CompactTextString(m) }
func (*PolicyTopicEntryTypeEnum) ProtoMessage()    {}
func (*PolicyTopicEntryTypeEnum) Descriptor() ([]byte, []int) {
	return fileDescriptor_policy_topic_entry_type_f2b35797c6ea84c5, []int{0}
}
func (m *PolicyTopicEntryTypeEnum) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PolicyTopicEntryTypeEnum.Unmarshal(m, b)
}
func (m *PolicyTopicEntryTypeEnum) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PolicyTopicEntryTypeEnum.Marshal(b, m, deterministic)
}
func (dst *PolicyTopicEntryTypeEnum) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PolicyTopicEntryTypeEnum.Merge(dst, src)
}
func (m *PolicyTopicEntryTypeEnum) XXX_Size() int {
	return xxx_messageInfo_PolicyTopicEntryTypeEnum.Size(m)
}
func (m *PolicyTopicEntryTypeEnum) XXX_DiscardUnknown() {
	xxx_messageInfo_PolicyTopicEntryTypeEnum.DiscardUnknown(m)
}

var xxx_messageInfo_PolicyTopicEntryTypeEnum proto.InternalMessageInfo

func init() {
	proto.RegisterType((*PolicyTopicEntryTypeEnum)(nil), "google.ads.googleads.v1.enums.PolicyTopicEntryTypeEnum")
	proto.RegisterEnum("google.ads.googleads.v1.enums.PolicyTopicEntryTypeEnum_PolicyTopicEntryType", PolicyTopicEntryTypeEnum_PolicyTopicEntryType_name, PolicyTopicEntryTypeEnum_PolicyTopicEntryType_value)
}

func init() {
	proto.RegisterFile("google/ads/googleads/v1/enums/policy_topic_entry_type.proto", fileDescriptor_policy_topic_entry_type_f2b35797c6ea84c5)
}

var fileDescriptor_policy_topic_entry_type_f2b35797c6ea84c5 = []byte{
	// 367 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0x41, 0x8a, 0xdb, 0x30,
	0x14, 0x86, 0x6b, 0xb7, 0x4d, 0x40, 0x81, 0xd6, 0x98, 0x16, 0x9a, 0xd2, 0x2c, 0x92, 0x03, 0xc8,
	0x98, 0xee, 0x94, 0x95, 0x1c, 0x2b, 0xa9, 0x68, 0x2a, 0x1b, 0xc7, 0x71, 0x69, 0x31, 0x18, 0x37,
	0x36, 0xc6, 0x90, 0x48, 0x26, 0x72, 0x02, 0x3e, 0x44, 0x2f, 0xd1, 0x65, 0x8e, 0xd2, 0xa3, 0x74,
	0xd1, 0x33, 0x0c, 0x92, 0x27, 0x59, 0x65, 0x66, 0x23, 0x7e, 0xe9, 0x7f, 0xdf, 0xcf, 0x7b, 0x4f,
	0x60, 0x5e, 0x09, 0x51, 0xed, 0x4b, 0x27, 0x2f, 0xa4, 0xd3, 0x4b, 0xa5, 0xce, 0xae, 0x53, 0xf2,
	0xd3, 0x41, 0x3a, 0x8d, 0xd8, 0xd7, 0xbb, 0x2e, 0x6b, 0x45, 0x53, 0xef, 0xb2, 0x92, 0xb7, 0xc7,
	0x2e, 0x6b, 0xbb, 0xa6, 0x84, 0xcd, 0x51, 0xb4, 0xc2, 0x9e, 0xf4, 0x04, 0xcc, 0x0b, 0x09, 0x6f,
	0x30, 0x3c, 0xbb, 0x50, 0xc3, 0x1f, 0x3f, 0x5d, 0xb3, 0x9b, 0xda, 0xc9, 0x39, 0x17, 0x6d, 0xde,
	0xd6, 0x82, 0xcb, 0x1e, 0x9e, 0x5d, 0x0c, 0xf0, 0x21, 0xd4, 0xf1, 0xb1, 0x4a, 0x27, 0x2a, 0x3c,
	0xee, 0x9a, 0x92, 0xf0, 0xd3, 0x61, 0xf6, 0xdb, 0x00, 0xef, 0xee, 0x99, 0xf6, 0x5b, 0x30, 0xda,
	0xb2, 0x4d, 0x48, 0x16, 0x74, 0x49, 0x89, 0x6f, 0xbd, 0xb0, 0x47, 0x60, 0xb8, 0x65, 0x5f, 0x59,
	0xf0, 0x9d, 0x59, 0x86, 0xfd, 0x06, 0x80, 0x30, 0x0a, 0xbe, 0x50, 0x8f, 0xc6, 0xc4, 0xb7, 0x4c,
	0x65, 0xae, 0xe9, 0x37, 0x7d, 0x79, 0xa5, 0x50, 0x9f, 0x6c, 0x16, 0x11, 0x0d, 0x63, 0x9a, 0x10,
	0xeb, 0xb5, 0xaa, 0xf6, 0xa2, 0x00, 0xfb, 0x84, 0x51, 0xb6, 0xb2, 0x06, 0xf6, 0x18, 0xbc, 0xc7,
	0x11, 0xc1, 0x59, 0xb0, 0xcc, 0x28, 0x8b, 0x49, 0x44, 0x36, 0x71, 0x16, 0xb0, 0xf5, 0x0f, 0x6b,
	0xe8, 0xfd, 0x37, 0xc0, 0x74, 0x27, 0x0e, 0xf0, 0xd9, 0x81, 0xbd, 0xf1, 0xbd, 0x96, 0x43, 0x35,
	0x6d, 0x68, 0xfc, 0xf4, 0x1e, 0xd9, 0x4a, 0xec, 0x73, 0x5e, 0x41, 0x71, 0xac, 0x9c, 0xaa, 0xe4,
	0x7a, 0x17, 0xd7, 0xcd, 0x37, 0xb5, 0x7c, 0xe2, 0x23, 0xe6, 0xfa, 0xfc, 0x63, 0xbe, 0x5c, 0x61,
	0x7c, 0x31, 0x27, 0xab, 0x3e, 0x0a, 0x17, 0x12, 0xf6, 0x52, 0xa9, 0xc4, 0x85, 0x6a, 0x77, 0xf2,
	0xef, 0xd5, 0x4f, 0x71, 0x21, 0xd3, 0x9b, 0x9f, 0x26, 0x6e, 0xaa, 0xfd, 0x7f, 0xe6, 0xb4, 0x7f,
	0x44, 0x08, 0x17, 0x12, 0xa1, 0x5b, 0x05, 0x42, 0x89, 0x8b, 0x90, 0xae, 0xf9, 0x35, 0xd0, 0x8d,
	0x7d, 0x7e, 0x08, 0x00, 0x00, 0xff, 0xff, 0xdd, 0xe2, 0xd2, 0x1b, 0x20, 0x02, 0x00, 0x00,
}
