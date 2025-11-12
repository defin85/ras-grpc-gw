package interceptor

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Helper function to create a benchmark message with password fields
func createBenchmarkMessage(clusterPassword, dbPassword, infobasePassword string) proto.Message {
	msgDesc := &descriptorpb.DescriptorProto{
		Name: proto.String("BenchmarkMessage"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("cluster_password"),
				Number: proto.Int32(1),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
			{
				Name:   proto.String("db_password"),
				Number: proto.Int32(2),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
			{
				Name:   proto.String("infobase_password"),
				Number: proto.Int32(3),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
			{
				Name:   proto.String("cluster_id"),
				Number: proto.Int32(4),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
			{
				Name:   proto.String("name"),
				Number: proto.Int32(5),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
		},
	}

	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("benchmark.proto"),
		Package:     proto.String("benchmark"),
		MessageType: []*descriptorpb.DescriptorProto{msgDesc},
	}

	file, err := protodesc.NewFile(fileDesc, nil)
	if err != nil {
		panic(err)
	}

	msgType := file.Messages().Get(0)
	msg := dynamicpb.NewMessage(msgType)

	msg.Set(msgType.Fields().ByName("cluster_password"), protoreflect.ValueOfString(clusterPassword))
	msg.Set(msgType.Fields().ByName("db_password"), protoreflect.ValueOfString(dbPassword))
	msg.Set(msgType.Fields().ByName("infobase_password"), protoreflect.ValueOfString(infobasePassword))
	msg.Set(msgType.Fields().ByName("cluster_id"), protoreflect.ValueOfString("cluster-123"))
	msg.Set(msgType.Fields().ByName("name"), protoreflect.ValueOfString("test-db"))

	return msg
}

// BenchmarkSanitizePasswordsInterceptor measures overhead of password sanitization
func BenchmarkSanitizePasswordsInterceptor(b *testing.B) {
	logger := zaptest.NewLogger(b)
	interceptor := SanitizePasswordsInterceptor(logger)

	// Create test request with 3 passwords
	req := createBenchmarkMessage("secret-cluster-pass", "secret-db-pass", "secret-ib-pass")

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

// BenchmarkSanitizePasswordsInterceptor_LargeMessage measures overhead for large messages
func BenchmarkSanitizePasswordsInterceptor_LargeMessage(b *testing.B) {
	logger := zaptest.NewLogger(b)
	interceptor := SanitizePasswordsInterceptor(logger)

	// Create large message (simulate realistic gRPC request)
	largeData := make([]byte, 1000)
	for i := range largeData {
		largeData[i] = byte('A' + (i % 26))
	}

	req := createBenchmarkMessage(
		"secret-pass-"+string(largeData[:100]),
		"secret-db-"+string(largeData[:100]),
		"secret-ib-"+string(largeData[:100]),
	)

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/infobase.service.InfobaseManagementService/UpdateInfobase",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, req, info, handler)
	}
}

// BenchmarkSanitizeMessage measures direct sanitization overhead
func BenchmarkSanitizeMessage(b *testing.B) {
	req := createBenchmarkMessage("secret-pass", "secret-db", "secret-ib")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeMessage(req)
	}
}

// BenchmarkSanitizeMessage_NoPasswords measures sanitization overhead for messages without passwords
func BenchmarkSanitizeMessage_NoPasswords(b *testing.B) {
	req := createBenchmarkMessage("", "", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeMessage(req)
	}
}

// BenchmarkSanitizePasswordInString measures password string sanitization
func BenchmarkSanitizePasswordInString(b *testing.B) {
	password := "very-secret-password-12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizePasswordInString(password)
	}
}
