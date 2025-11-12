package interceptor

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Helper function to create an audit benchmark message
func createAuditBenchmarkMessage(clusterID, infobaseID, user string) proto.Message {
	msgDesc := &descriptorpb.DescriptorProto{
		Name: proto.String("AuditBenchmarkMessage"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("cluster_id"),
				Number: proto.Int32(1),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
			{
				Name:   proto.String("infobase_id"),
				Number: proto.Int32(2),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
			{
				Name:   proto.String("cluster_user"),
				Number: proto.Int32(3),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
		},
	}

	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("audit_benchmark.proto"),
		Package:     proto.String("audit"),
		MessageType: []*descriptorpb.DescriptorProto{msgDesc},
	}

	file, err := protodesc.NewFile(fileDesc, nil)
	if err != nil {
		panic(err)
	}

	msgType := file.Messages().Get(0)
	msg := dynamicpb.NewMessage(msgType)

	msg.Set(msgType.Fields().ByName("cluster_id"), protoreflect.ValueOfString(clusterID))
	msg.Set(msgType.Fields().ByName("infobase_id"), protoreflect.ValueOfString(infobaseID))
	msg.Set(msgType.Fields().ByName("cluster_user"), protoreflect.ValueOfString(user))

	return msg
}

// BenchmarkAuditInterceptor measures overhead of audit logging
func BenchmarkAuditInterceptor(b *testing.B) {
	logger := zaptest.NewLogger(b)
	interceptor := AuditInterceptor(logger)

	req := createAuditBenchmarkMessage("cluster-123", "infobase-456", "admin")

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/infobase.service.InfobaseManagementService/CreateInfobase",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, req, info, handler)
	}
}

// BenchmarkAuditInterceptor_DestructiveOperation measures audit overhead for destructive ops
func BenchmarkAuditInterceptor_DestructiveOperation(b *testing.B) {
	logger := zaptest.NewLogger(b)
	interceptor := AuditInterceptor(logger)

	req := createAuditBenchmarkMessage("cluster-123", "infobase-to-drop", "admin")

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/infobase.service.InfobaseManagementService/DropInfobase",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, req, info, handler)
	}
}

// BenchmarkAuditInterceptor_WithError measures audit overhead when handler returns error
func BenchmarkAuditInterceptor_WithError(b *testing.B) {
	logger := zaptest.NewLogger(b)
	interceptor := AuditInterceptor(logger)

	req := createAuditBenchmarkMessage("cluster-123", "infobase-456", "admin")

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/infobase.service.InfobaseManagementService/CreateInfobase",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, errors.New("test error")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, req, info, handler)
	}
}

// BenchmarkExtractAuditMetadata measures metadata extraction overhead
func BenchmarkExtractAuditMetadata(b *testing.B) {
	req := createAuditBenchmarkMessage("cluster-123", "infobase-456", "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractAuditMetadata(req)
	}
}

// BenchmarkChainedInterceptors measures combined overhead of sanitize + audit
func BenchmarkChainedInterceptors(b *testing.B) {
	logger := zaptest.NewLogger(b)
	sanitizeInterceptor := SanitizePasswordsInterceptor(logger)
	auditInterceptor := AuditInterceptor(logger)

	req := createAuditBenchmarkMessage("cluster-123", "infobase-456", "admin")

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/infobase.service.InfobaseManagementService/CreateInfobase",
	}
	finalHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	// Chain interceptors: sanitize -> audit -> handler
	chainedHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return auditInterceptor(ctx, req, info, finalHandler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sanitizeInterceptor(ctx, req, info, chainedHandler)
	}
}
