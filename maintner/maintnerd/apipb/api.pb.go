// Code generated by protoc-gen-go4grpc; DO NOT EDIT
// source: api.proto

/*
Package apipb is a generated protocol buffer package.

It is generated from these files:
	api.proto

It has these top-level messages:
	HasAncestorRequest
	HasAncestorResponse
	GetRefRequest
	GetRefResponse
	GoFindTryWorkRequest
	GoFindTryWorkResponse
	GerritTryWorkItem
	TryVoteMessage
	MajorMinor
	ListGoReleasesRequest
	ListGoReleasesResponse
	GoRelease
*/
package apipb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "context"
	grpc "grpc.go4.org"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type HasAncestorRequest struct {
	Commit   string `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
	Ancestor string `protobuf:"bytes,2,opt,name=ancestor" json:"ancestor,omitempty"`
}

func (m *HasAncestorRequest) Reset()                    { *m = HasAncestorRequest{} }
func (m *HasAncestorRequest) String() string            { return proto.CompactTextString(m) }
func (*HasAncestorRequest) ProtoMessage()               {}
func (*HasAncestorRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *HasAncestorRequest) GetCommit() string {
	if m != nil {
		return m.Commit
	}
	return ""
}

func (m *HasAncestorRequest) GetAncestor() string {
	if m != nil {
		return m.Ancestor
	}
	return ""
}

type HasAncestorResponse struct {
	// has_ancestor is whether ancestor appears in commit's history.
	HasAncestor bool `protobuf:"varint,1,opt,name=has_ancestor,json=hasAncestor" json:"has_ancestor,omitempty"`
	// unknown_commit is true if the provided commit was unknown.
	UnknownCommit bool `protobuf:"varint,2,opt,name=unknown_commit,json=unknownCommit" json:"unknown_commit,omitempty"`
}

func (m *HasAncestorResponse) Reset()                    { *m = HasAncestorResponse{} }
func (m *HasAncestorResponse) String() string            { return proto.CompactTextString(m) }
func (*HasAncestorResponse) ProtoMessage()               {}
func (*HasAncestorResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *HasAncestorResponse) GetHasAncestor() bool {
	if m != nil {
		return m.HasAncestor
	}
	return false
}

func (m *HasAncestorResponse) GetUnknownCommit() bool {
	if m != nil {
		return m.UnknownCommit
	}
	return false
}

type GetRefRequest struct {
	Ref string `protobuf:"bytes,1,opt,name=ref" json:"ref,omitempty"`
	// Either gerrit_server & gerrit_project must be specified, or
	// github. Currently only Gerrit is supported.
	GerritServer  string `protobuf:"bytes,2,opt,name=gerrit_server,json=gerritServer" json:"gerrit_server,omitempty"`
	GerritProject string `protobuf:"bytes,3,opt,name=gerrit_project,json=gerritProject" json:"gerrit_project,omitempty"`
}

func (m *GetRefRequest) Reset()                    { *m = GetRefRequest{} }
func (m *GetRefRequest) String() string            { return proto.CompactTextString(m) }
func (*GetRefRequest) ProtoMessage()               {}
func (*GetRefRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *GetRefRequest) GetRef() string {
	if m != nil {
		return m.Ref
	}
	return ""
}

func (m *GetRefRequest) GetGerritServer() string {
	if m != nil {
		return m.GerritServer
	}
	return ""
}

func (m *GetRefRequest) GetGerritProject() string {
	if m != nil {
		return m.GerritProject
	}
	return ""
}

type GetRefResponse struct {
	Value string `protobuf:"bytes,1,opt,name=value" json:"value,omitempty"`
}

func (m *GetRefResponse) Reset()                    { *m = GetRefResponse{} }
func (m *GetRefResponse) String() string            { return proto.CompactTextString(m) }
func (*GetRefResponse) ProtoMessage()               {}
func (*GetRefResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *GetRefResponse) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

type GoFindTryWorkRequest struct {
	// for_staging says whether this is a trybot request for the staging
	// cluster. When using staging, the comment "Run-StagingTryBot"
	// is used instead of label:Run-TryBot=1.
	ForStaging bool `protobuf:"varint,1,opt,name=for_staging,json=forStaging" json:"for_staging,omitempty"`
}

func (m *GoFindTryWorkRequest) Reset()                    { *m = GoFindTryWorkRequest{} }
func (m *GoFindTryWorkRequest) String() string            { return proto.CompactTextString(m) }
func (*GoFindTryWorkRequest) ProtoMessage()               {}
func (*GoFindTryWorkRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *GoFindTryWorkRequest) GetForStaging() bool {
	if m != nil {
		return m.ForStaging
	}
	return false
}

type GoFindTryWorkResponse struct {
	// waiting are the Gerrit CLs wanting a trybot run and not yet with results.
	// These might already be running.
	Waiting []*GerritTryWorkItem `protobuf:"bytes,1,rep,name=waiting" json:"waiting,omitempty"`
}

func (m *GoFindTryWorkResponse) Reset()                    { *m = GoFindTryWorkResponse{} }
func (m *GoFindTryWorkResponse) String() string            { return proto.CompactTextString(m) }
func (*GoFindTryWorkResponse) ProtoMessage()               {}
func (*GoFindTryWorkResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *GoFindTryWorkResponse) GetWaiting() []*GerritTryWorkItem {
	if m != nil {
		return m.Waiting
	}
	return nil
}

type GerritTryWorkItem struct {
	Project  string `protobuf:"bytes,1,opt,name=project" json:"project,omitempty"`
	Branch   string `protobuf:"bytes,2,opt,name=branch" json:"branch,omitempty"`
	ChangeId string `protobuf:"bytes,3,opt,name=change_id,json=changeId" json:"change_id,omitempty"`
	Commit   string `protobuf:"bytes,4,opt,name=commit" json:"commit,omitempty"`
	Version  int32  `protobuf:"varint,9,opt,name=version" json:"version,omitempty"`
	// go_commit is set for subrepos and is the Go commit(s) to test against.
	// go_branch is a branch name of go_commit, for showing to users when
	// a try set fails.
	GoCommit []string `protobuf:"bytes,5,rep,name=go_commit,json=goCommit" json:"go_commit,omitempty"`
	GoBranch []string `protobuf:"bytes,6,rep,name=go_branch,json=goBranch" json:"go_branch,omitempty"`
	// go_version specifies the major and minor version of the targeted Go toolchain.
	// For Go repo, it contains exactly one element.
	// For subrepos, it contains elements that correspond to go_commit.
	GoVersion []*MajorMinor `protobuf:"bytes,7,rep,name=go_version,json=goVersion" json:"go_version,omitempty"`
	// try_message is the list of TRY=xxxx messages associated with Run-TryBot votes.
	TryMessage []*TryVoteMessage `protobuf:"bytes,8,rep,name=try_message,json=tryMessage" json:"try_message,omitempty"`
}

func (m *GerritTryWorkItem) Reset()                    { *m = GerritTryWorkItem{} }
func (m *GerritTryWorkItem) String() string            { return proto.CompactTextString(m) }
func (*GerritTryWorkItem) ProtoMessage()               {}
func (*GerritTryWorkItem) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *GerritTryWorkItem) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *GerritTryWorkItem) GetBranch() string {
	if m != nil {
		return m.Branch
	}
	return ""
}

func (m *GerritTryWorkItem) GetChangeId() string {
	if m != nil {
		return m.ChangeId
	}
	return ""
}

func (m *GerritTryWorkItem) GetCommit() string {
	if m != nil {
		return m.Commit
	}
	return ""
}

func (m *GerritTryWorkItem) GetVersion() int32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *GerritTryWorkItem) GetGoCommit() []string {
	if m != nil {
		return m.GoCommit
	}
	return nil
}

func (m *GerritTryWorkItem) GetGoBranch() []string {
	if m != nil {
		return m.GoBranch
	}
	return nil
}

func (m *GerritTryWorkItem) GetGoVersion() []*MajorMinor {
	if m != nil {
		return m.GoVersion
	}
	return nil
}

func (m *GerritTryWorkItem) GetTryMessage() []*TryVoteMessage {
	if m != nil {
		return m.TryMessage
	}
	return nil
}

type TryVoteMessage struct {
	Message  string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
	AuthorId int64  `protobuf:"varint,2,opt,name=author_id,json=authorId" json:"author_id,omitempty"`
	Version  int32  `protobuf:"varint,3,opt,name=version" json:"version,omitempty"`
}

func (m *TryVoteMessage) Reset()                    { *m = TryVoteMessage{} }
func (m *TryVoteMessage) String() string            { return proto.CompactTextString(m) }
func (*TryVoteMessage) ProtoMessage()               {}
func (*TryVoteMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *TryVoteMessage) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *TryVoteMessage) GetAuthorId() int64 {
	if m != nil {
		return m.AuthorId
	}
	return 0
}

func (m *TryVoteMessage) GetVersion() int32 {
	if m != nil {
		return m.Version
	}
	return 0
}

type MajorMinor struct {
	Major int32 `protobuf:"varint,1,opt,name=major" json:"major,omitempty"`
	Minor int32 `protobuf:"varint,2,opt,name=minor" json:"minor,omitempty"`
}

func (m *MajorMinor) Reset()                    { *m = MajorMinor{} }
func (m *MajorMinor) String() string            { return proto.CompactTextString(m) }
func (*MajorMinor) ProtoMessage()               {}
func (*MajorMinor) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *MajorMinor) GetMajor() int32 {
	if m != nil {
		return m.Major
	}
	return 0
}

func (m *MajorMinor) GetMinor() int32 {
	if m != nil {
		return m.Minor
	}
	return 0
}

type ListGoReleasesRequest struct {
}

func (m *ListGoReleasesRequest) Reset()                    { *m = ListGoReleasesRequest{} }
func (m *ListGoReleasesRequest) String() string            { return proto.CompactTextString(m) }
func (*ListGoReleasesRequest) ProtoMessage()               {}
func (*ListGoReleasesRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

type ListGoReleasesResponse struct {
	Releases []*GoRelease `protobuf:"bytes,1,rep,name=releases" json:"releases,omitempty"`
}

func (m *ListGoReleasesResponse) Reset()                    { *m = ListGoReleasesResponse{} }
func (m *ListGoReleasesResponse) String() string            { return proto.CompactTextString(m) }
func (*ListGoReleasesResponse) ProtoMessage()               {}
func (*ListGoReleasesResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func (m *ListGoReleasesResponse) GetReleases() []*GoRelease {
	if m != nil {
		return m.Releases
	}
	return nil
}

type GoRelease struct {
	Major     int32  `protobuf:"varint,1,opt,name=major" json:"major,omitempty"`
	Minor     int32  `protobuf:"varint,2,opt,name=minor" json:"minor,omitempty"`
	Patch     int32  `protobuf:"varint,3,opt,name=patch" json:"patch,omitempty"`
	TagName   string `protobuf:"bytes,4,opt,name=tag_name,json=tagName" json:"tag_name,omitempty"`
	TagCommit string `protobuf:"bytes,5,opt,name=tag_commit,json=tagCommit" json:"tag_commit,omitempty"`
	// Release branch information for this major-minor version pair.
	BranchName   string `protobuf:"bytes,6,opt,name=branch_name,json=branchName" json:"branch_name,omitempty"`
	BranchCommit string `protobuf:"bytes,7,opt,name=branch_commit,json=branchCommit" json:"branch_commit,omitempty"`
}

func (m *GoRelease) Reset()                    { *m = GoRelease{} }
func (m *GoRelease) String() string            { return proto.CompactTextString(m) }
func (*GoRelease) ProtoMessage()               {}
func (*GoRelease) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{11} }

func (m *GoRelease) GetMajor() int32 {
	if m != nil {
		return m.Major
	}
	return 0
}

func (m *GoRelease) GetMinor() int32 {
	if m != nil {
		return m.Minor
	}
	return 0
}

func (m *GoRelease) GetPatch() int32 {
	if m != nil {
		return m.Patch
	}
	return 0
}

func (m *GoRelease) GetTagName() string {
	if m != nil {
		return m.TagName
	}
	return ""
}

func (m *GoRelease) GetTagCommit() string {
	if m != nil {
		return m.TagCommit
	}
	return ""
}

func (m *GoRelease) GetBranchName() string {
	if m != nil {
		return m.BranchName
	}
	return ""
}

func (m *GoRelease) GetBranchCommit() string {
	if m != nil {
		return m.BranchCommit
	}
	return ""
}

func init() {
	proto.RegisterType((*HasAncestorRequest)(nil), "apipb.HasAncestorRequest")
	proto.RegisterType((*HasAncestorResponse)(nil), "apipb.HasAncestorResponse")
	proto.RegisterType((*GetRefRequest)(nil), "apipb.GetRefRequest")
	proto.RegisterType((*GetRefResponse)(nil), "apipb.GetRefResponse")
	proto.RegisterType((*GoFindTryWorkRequest)(nil), "apipb.GoFindTryWorkRequest")
	proto.RegisterType((*GoFindTryWorkResponse)(nil), "apipb.GoFindTryWorkResponse")
	proto.RegisterType((*GerritTryWorkItem)(nil), "apipb.GerritTryWorkItem")
	proto.RegisterType((*TryVoteMessage)(nil), "apipb.TryVoteMessage")
	proto.RegisterType((*MajorMinor)(nil), "apipb.MajorMinor")
	proto.RegisterType((*ListGoReleasesRequest)(nil), "apipb.ListGoReleasesRequest")
	proto.RegisterType((*ListGoReleasesResponse)(nil), "apipb.ListGoReleasesResponse")
	proto.RegisterType((*GoRelease)(nil), "apipb.GoRelease")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for MaintnerService service

type MaintnerServiceClient interface {
	// HasAncestor reports whether one commit contains another commit
	// in its git history.
	HasAncestor(ctx context.Context, in *HasAncestorRequest, opts ...grpc.CallOption) (*HasAncestorResponse, error)
	// GetRef returns information about a git ref.
	GetRef(ctx context.Context, in *GetRefRequest, opts ...grpc.CallOption) (*GetRefResponse, error)
	// GoFindTryWork finds trybot work for the coordinator to build & test.
	GoFindTryWork(ctx context.Context, in *GoFindTryWorkRequest, opts ...grpc.CallOption) (*GoFindTryWorkResponse, error)
	// ListGoReleases lists Go releases sorted by version with latest first.
	//
	// A release is considered to exist for each git tag named "goX", "goX.Y", or
	// "goX.Y.Z", as long as it has a corresponding "release-branch.goX" or
	// "release-branch.goX.Y" release branch.
	//
	// ListGoReleases returns only the latest patch versions of releases which
	// are considered supported per policy. For example, Go 1.12.6 and 1.11.11.
	// The response is guaranteed to have two versions, otherwise an error
	// is returned.
	ListGoReleases(ctx context.Context, in *ListGoReleasesRequest, opts ...grpc.CallOption) (*ListGoReleasesResponse, error)
}

type maintnerServiceClient struct {
	cc *grpc.ClientConn
}

func NewMaintnerServiceClient(cc *grpc.ClientConn) MaintnerServiceClient {
	return &maintnerServiceClient{cc}
}

func (c *maintnerServiceClient) HasAncestor(ctx context.Context, in *HasAncestorRequest, opts ...grpc.CallOption) (*HasAncestorResponse, error) {
	out := new(HasAncestorResponse)
	err := grpc.Invoke(ctx, "/apipb.MaintnerService/HasAncestor", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *maintnerServiceClient) GetRef(ctx context.Context, in *GetRefRequest, opts ...grpc.CallOption) (*GetRefResponse, error) {
	out := new(GetRefResponse)
	err := grpc.Invoke(ctx, "/apipb.MaintnerService/GetRef", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *maintnerServiceClient) GoFindTryWork(ctx context.Context, in *GoFindTryWorkRequest, opts ...grpc.CallOption) (*GoFindTryWorkResponse, error) {
	out := new(GoFindTryWorkResponse)
	err := grpc.Invoke(ctx, "/apipb.MaintnerService/GoFindTryWork", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *maintnerServiceClient) ListGoReleases(ctx context.Context, in *ListGoReleasesRequest, opts ...grpc.CallOption) (*ListGoReleasesResponse, error) {
	out := new(ListGoReleasesResponse)
	err := grpc.Invoke(ctx, "/apipb.MaintnerService/ListGoReleases", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for MaintnerService service

type MaintnerServiceServer interface {
	// HasAncestor reports whether one commit contains another commit
	// in its git history.
	HasAncestor(context.Context, *HasAncestorRequest) (*HasAncestorResponse, error)
	// GetRef returns information about a git ref.
	GetRef(context.Context, *GetRefRequest) (*GetRefResponse, error)
	// GoFindTryWork finds trybot work for the coordinator to build & test.
	GoFindTryWork(context.Context, *GoFindTryWorkRequest) (*GoFindTryWorkResponse, error)
	// ListGoReleases lists Go releases sorted by version with latest first.
	//
	// A release is considered to exist for each git tag named "goX", "goX.Y", or
	// "goX.Y.Z", as long as it has a corresponding "release-branch.goX" or
	// "release-branch.goX.Y" release branch.
	//
	// ListGoReleases returns only the latest patch versions of releases which
	// are considered supported per policy. For example, Go 1.12.6 and 1.11.11.
	// The response is guaranteed to have two versions, otherwise an error
	// is returned.
	ListGoReleases(context.Context, *ListGoReleasesRequest) (*ListGoReleasesResponse, error)
}

func RegisterMaintnerServiceServer(s *grpc.Server, srv MaintnerServiceServer) {
	s.RegisterService(&_MaintnerService_serviceDesc, srv)
}

func _MaintnerService_HasAncestor_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HasAncestorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MaintnerServiceServer).HasAncestor(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apipb.MaintnerService/HasAncestor",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MaintnerServiceServer).HasAncestor(ctx, req.(*HasAncestorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MaintnerService_GetRef_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRefRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MaintnerServiceServer).GetRef(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apipb.MaintnerService/GetRef",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MaintnerServiceServer).GetRef(ctx, req.(*GetRefRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MaintnerService_GoFindTryWork_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GoFindTryWorkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MaintnerServiceServer).GoFindTryWork(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apipb.MaintnerService/GoFindTryWork",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MaintnerServiceServer).GoFindTryWork(ctx, req.(*GoFindTryWorkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MaintnerService_ListGoReleases_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListGoReleasesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MaintnerServiceServer).ListGoReleases(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/apipb.MaintnerService/ListGoReleases",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MaintnerServiceServer).ListGoReleases(ctx, req.(*ListGoReleasesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _MaintnerService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "apipb.MaintnerService",
	HandlerType: (*MaintnerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "HasAncestor",
			Handler:    _MaintnerService_HasAncestor_Handler,
		},
		{
			MethodName: "GetRef",
			Handler:    _MaintnerService_GetRef_Handler,
		},
		{
			MethodName: "GoFindTryWork",
			Handler:    _MaintnerService_GoFindTryWork_Handler,
		},
		{
			MethodName: "ListGoReleases",
			Handler:    _MaintnerService_ListGoReleases_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.proto",
}

func init() { proto.RegisterFile("api.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 700 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x55, 0xdb, 0x4e, 0x1b, 0x31,
	0x10, 0x55, 0x92, 0xe6, 0x36, 0x21, 0x29, 0xb8, 0x84, 0x2e, 0xa1, 0x08, 0xba, 0xa8, 0x15, 0x0f,
	0x15, 0xaa, 0xa8, 0x7a, 0x79, 0xed, 0x45, 0x04, 0xda, 0xa6, 0xaa, 0x16, 0x44, 0x1f, 0x57, 0x26,
	0x71, 0x36, 0x0b, 0xac, 0xbd, 0xb5, 0x1d, 0x50, 0x3e, 0xa9, 0x7f, 0xd0, 0x9f, 0xe8, 0x3f, 0x55,
	0xb6, 0xc7, 0x4b, 0x02, 0xf4, 0xa1, 0x6f, 0x3b, 0xe7, 0xcc, 0x1c, 0x8f, 0xc7, 0xc7, 0x5e, 0x68,
	0xd2, 0x3c, 0xdd, 0xcb, 0xa5, 0xd0, 0x82, 0x54, 0x69, 0x9e, 0xe6, 0x67, 0xe1, 0x21, 0x90, 0x43,
	0xaa, 0xde, 0xf3, 0x21, 0x53, 0x5a, 0xc8, 0x88, 0xfd, 0x9c, 0x32, 0xa5, 0xc9, 0x1a, 0xd4, 0x86,
	0x22, 0xcb, 0x52, 0x1d, 0x94, 0xb6, 0x4b, 0xbb, 0xcd, 0x08, 0x23, 0xd2, 0x83, 0x06, 0xc5, 0xd4,
	0xa0, 0x6c, 0x99, 0x22, 0x0e, 0x63, 0x78, 0xb4, 0xa0, 0xa4, 0x72, 0xc1, 0x15, 0x23, 0x4f, 0x61,
	0x69, 0x42, 0x55, 0x5c, 0x94, 0x19, 0xc1, 0x46, 0xd4, 0x9a, 0xdc, 0xa4, 0x92, 0x67, 0xd0, 0x99,
	0xf2, 0x0b, 0x2e, 0xae, 0x79, 0x8c, 0xab, 0x96, 0x6d, 0x52, 0x1b, 0xd1, 0x8f, 0x16, 0x0c, 0x33,
	0x68, 0xf7, 0x99, 0x8e, 0xd8, 0xd8, 0x77, 0xb9, 0x0c, 0x15, 0xc9, 0xc6, 0xd8, 0xa2, 0xf9, 0x24,
	0x3b, 0xd0, 0x4e, 0x98, 0x94, 0xa9, 0x8e, 0x15, 0x93, 0x57, 0xcc, 0x37, 0xb9, 0xe4, 0xc0, 0x63,
	0x8b, 0x99, 0xe5, 0x30, 0x29, 0x97, 0xe2, 0x9c, 0x0d, 0x75, 0x50, 0xb1, 0x59, 0x58, 0xfa, 0xdd,
	0x81, 0xe1, 0x73, 0xe8, 0xf8, 0xe5, 0x70, 0x2b, 0xab, 0x50, 0xbd, 0xa2, 0x97, 0x53, 0x86, 0x2b,
	0xba, 0x20, 0x7c, 0x0b, 0xab, 0x7d, 0x71, 0x90, 0xf2, 0xd1, 0x89, 0x9c, 0xfd, 0x10, 0xf2, 0xc2,
	0x77, 0xb7, 0x05, 0xad, 0xb1, 0x90, 0xb1, 0xd2, 0x34, 0x49, 0x79, 0x82, 0xfb, 0x86, 0xb1, 0x90,
	0xc7, 0x0e, 0x09, 0xbf, 0x40, 0xf7, 0x56, 0x21, 0xae, 0xb3, 0x0f, 0xf5, 0x6b, 0x9a, 0x6a, 0x57,
	0x55, 0xd9, 0x6d, 0xed, 0x07, 0x7b, 0xf6, 0xb0, 0xf6, 0xfa, 0xb6, 0x41, 0x4c, 0x3f, 0xd2, 0x2c,
	0x8b, 0x7c, 0x62, 0xf8, 0xbb, 0x0c, 0x2b, 0x77, 0x68, 0x12, 0x40, 0xdd, 0xef, 0xd1, 0xf5, 0xec,
	0x43, 0x73, 0xc2, 0x67, 0x92, 0xf2, 0xe1, 0x04, 0x47, 0x84, 0x11, 0xd9, 0x80, 0xe6, 0x70, 0x42,
	0x79, 0xc2, 0xe2, 0x74, 0x84, 0x73, 0x69, 0x38, 0xe0, 0x68, 0x34, 0x67, 0x8b, 0x07, 0x0b, 0xb6,
	0x08, 0xa0, 0x7e, 0xc5, 0xa4, 0x4a, 0x05, 0x0f, 0x9a, 0xdb, 0xa5, 0xdd, 0x6a, 0xe4, 0x43, 0x23,
	0x97, 0x08, 0x7f, 0xaa, 0xd5, 0xed, 0x8a, 0x91, 0x4b, 0x84, 0x3b, 0x50, 0x24, 0xb1, 0x8d, 0x9a,
	0x27, 0x3f, 0xb8, 0x46, 0x5e, 0x02, 0x24, 0x22, 0xf6, 0xb2, 0x75, 0x3b, 0x87, 0x15, 0x9c, 0xc3,
	0x80, 0x9e, 0x0b, 0x39, 0x48, 0xb9, 0x90, 0x51, 0x33, 0x11, 0xa7, 0xb8, 0xd6, 0x1b, 0x68, 0x69,
	0x39, 0x8b, 0x33, 0xa6, 0x14, 0x4d, 0x58, 0xd0, 0xb0, 0x25, 0x5d, 0x2c, 0x39, 0x91, 0xb3, 0x53,
	0xa1, 0xd9, 0xc0, 0x91, 0x11, 0x68, 0x39, 0xc3, 0xef, 0x90, 0x42, 0x67, 0x91, 0x35, 0xfb, 0xf1,
	0x2a, 0x38, 0x36, 0x0c, 0x4d, 0xcb, 0x74, 0xaa, 0x27, 0x42, 0x9a, 0xf1, 0x98, 0xc9, 0x55, 0xa2,
	0x86, 0x03, 0x8e, 0x46, 0xf3, 0x63, 0xa8, 0x2c, 0x8c, 0x21, 0x7c, 0x07, 0x70, 0xd3, 0xb3, 0xf1,
	0x51, 0x66, 0x22, 0x2b, 0x5e, 0x8d, 0x5c, 0x60, 0x51, 0x43, 0x5b, 0x59, 0x83, 0x9a, 0x20, 0x7c,
	0x0c, 0xdd, 0xaf, 0xa9, 0xd2, 0x7d, 0x11, 0xb1, 0x4b, 0x46, 0x15, 0x53, 0x68, 0xaf, 0xf0, 0x00,
	0xd6, 0x6e, 0x13, 0x68, 0x9f, 0x17, 0xd0, 0x90, 0x88, 0xa1, 0x7f, 0x96, 0xbd, 0x7f, 0x7c, 0x72,
	0x54, 0x64, 0x84, 0x7f, 0x4a, 0xd0, 0x2c, 0xf0, 0xff, 0x69, 0xcd, 0xa0, 0x39, 0xd5, 0xc3, 0x09,
	0x6e, 0xd6, 0x05, 0x64, 0x1d, 0x1a, 0x9a, 0x26, 0x31, 0xa7, 0x19, 0x43, 0x97, 0xd4, 0x35, 0x4d,
	0xbe, 0xd1, 0x8c, 0x91, 0x4d, 0x00, 0x43, 0x15, 0x6e, 0x30, 0x64, 0x53, 0xd3, 0x04, 0xed, 0xb0,
	0x05, 0x2d, 0xe7, 0x05, 0x57, 0x5c, 0xb3, 0x3c, 0x38, 0xc8, 0xd6, 0xef, 0x40, 0x1b, 0x13, 0x50,
	0xa2, 0xee, 0x6e, 0xb7, 0x03, 0x9d, 0xca, 0xfe, 0xaf, 0x32, 0x3c, 0x1c, 0xd0, 0x94, 0x6b, 0xce,
	0xa4, 0xb9, 0xf0, 0xe9, 0x90, 0x91, 0x4f, 0xd0, 0x9a, 0x7b, 0x9a, 0xc8, 0x3a, 0x8e, 0xe3, 0xee,
	0xc3, 0xd7, 0xeb, 0xdd, 0x47, 0xe1, 0x5c, 0x5f, 0x43, 0xcd, 0x3d, 0x08, 0x64, 0xb5, 0xb8, 0x8f,
	0x73, 0xcf, 0x51, 0xaf, 0x7b, 0x0b, 0xc5, 0xb2, 0xcf, 0xd0, 0x5e, 0xb8, 0xe6, 0x64, 0xa3, 0x38,
	0x8d, 0xbb, 0xaf, 0x46, 0xef, 0xc9, 0xfd, 0x24, 0x6a, 0x0d, 0xa0, 0xb3, 0x78, 0xe8, 0xc4, 0xe7,
	0xdf, 0x6b, 0x92, 0xde, 0xe6, 0x3f, 0x58, 0x27, 0x77, 0x56, 0xb3, 0xbf, 0x82, 0x57, 0x7f, 0x03,
	0x00, 0x00, 0xff, 0xff, 0x0d, 0x79, 0xa6, 0xa3, 0x17, 0x06, 0x00, 0x00,
}
