package interceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Mock gRPC handler for testing
func mockHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "response", nil
}

// Mock gRPC server info
func mockServerInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{
		FullMethod: method,
	}
}

// Helper function to create a test proto message with password fields
func createTestMessage(clusterPassword, dbPassword, infobasePassword string) proto.Message {
	msgDesc := &descriptorpb.DescriptorProto{
		Name: proto.String("TestMessage"),
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
				Name:   proto.String("name"),
				Number: proto.Int32(4),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
		},
	}

	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("test.proto"),
		Package:     proto.String("test"),
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
	msg.Set(msgType.Fields().ByName("name"), protoreflect.ValueOfString("test-infobase"))

	return msg
}

func TestSanitizePasswordsInterceptor_AllPasswordFields(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	req := createTestMessage("secret1", "secret2", "secret3")

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	reflectMsg := req.ProtoReflect()
	msgType := reflectMsg.Type().Descriptor()

	clusterPwd := reflectMsg.Get(msgType.Fields().ByName("cluster_password")).String()
	dbPwd := reflectMsg.Get(msgType.Fields().ByName("db_password")).String()
	infobasePwd := reflectMsg.Get(msgType.Fields().ByName("infobase_password")).String()

	assert.Equal(t, "secret1", clusterPwd, "cluster_password should not be modified")
	assert.Equal(t, "secret2", dbPwd, "db_password should not be modified")
	assert.Equal(t, "secret3", infobasePwd, "infobase_password should not be modified")
}

func TestSanitizePasswordsInterceptor_EmptyPasswords(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	req := createTestMessage("", "", "")

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	reflectMsg := req.ProtoReflect()
	msgType := reflectMsg.Type().Descriptor()

	clusterPwd := reflectMsg.Get(msgType.Fields().ByName("cluster_password")).String()
	dbPwd := reflectMsg.Get(msgType.Fields().ByName("db_password")).String()
	infobasePwd := reflectMsg.Get(msgType.Fields().ByName("infobase_password")).String()

	assert.Equal(t, "", clusterPwd)
	assert.Equal(t, "", dbPwd)
	assert.Equal(t, "", infobasePwd)
}

func TestSanitizePasswordInString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"non-empty password", "secret123", passwordMask},
		{"empty password", "", ""},
		{"single character", "x", passwordMask},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePasswordInString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePasswordsStreamInterceptor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsStreamInterceptor(logger)

	mockStreamHandler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.Service/StreamMethod",
		IsClientStream: true,
		IsServerStream: true,
	}

	err := interceptor(nil, nil, info, mockStreamHandler)
	assert.NoError(t, err)
}

func TestSanitizePasswordsInterceptor_NonProtoMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	req := "not a proto message"

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)
}
