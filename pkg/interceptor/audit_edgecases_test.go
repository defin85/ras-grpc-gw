package interceptor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Helper to create a test proto message with audit metadata
func createAuditTestMessage(clusterID, infobaseID, clusterUser string) proto.Message {
	msgDesc := &descriptorpb.DescriptorProto{
		Name: proto.String("AuditTestMessage"),
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
		Name:        proto.String("audit_test.proto"),
		Package:     proto.String("test"),
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
	msg.Set(msgType.Fields().ByName("cluster_user"), protoreflect.ValueOfString(clusterUser))

	return msg
}

// TestAuditInterceptor_DestructiveOperation tests that DropInfobase is logged as WARN
func TestAuditInterceptor_DestructiveOperation(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditInterceptor(logger)

	req := createAuditTestMessage("cluster-123", "infobase-456", "admin")

	ctx := context.Background()
	info := mockServerInfo("/infobase.service.InfobaseManagementService/DropInfobase")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Check logs
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	assert.Equal(t, zapcore.WarnLevel, logEntry.Level, "DropInfobase should be WARN level")
	assert.Equal(t, "gRPC destructive operation", logEntry.Message)
	assert.Equal(t, "/infobase.service.InfobaseManagementService/DropInfobase", logEntry.ContextMap()["operation"])
}

// TestAuditInterceptor_ErrorLogging tests that errors are logged as ERROR
func TestAuditInterceptor_ErrorLogging(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditInterceptor(logger)

	// Mock handler that returns error
	errorHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.NotFound, "infobase not found")
	}

	req := createAuditTestMessage("cluster-123", "infobase-456", "admin")

	ctx := context.Background()
	info := mockServerInfo("/infobase.service.InfobaseManagementService/UpdateInfobase")

	resp, err := interceptor(ctx, req, info, errorHandler)

	require.Error(t, err)
	assert.Nil(t, resp)

	// Check logs
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	assert.Equal(t, zapcore.ErrorLevel, logEntry.Level, "Failed operation should be ERROR level")
	assert.Equal(t, "gRPC operation failed", logEntry.Message)
	assert.Equal(t, "error", logEntry.ContextMap()["result"])
	assert.Equal(t, "NotFound", logEntry.ContextMap()["grpc_code"])
}

// TestAuditInterceptor_MissingMetadata tests audit with missing metadata fields
func TestAuditInterceptor_MissingMetadata(t *testing.T) {
	tests := []struct {
		name        string
		clusterID   string
		infobaseID  string
		clusterUser string
	}{
		{"all missing", "", "", ""},
		{"cluster missing", "", "infobase-456", "admin"},
		{"infobase missing", "cluster-123", "", "admin"},
		{"user missing", "cluster-123", "infobase-456", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)
			interceptor := AuditInterceptor(logger)

			req := createAuditTestMessage(tt.clusterID, tt.infobaseID, tt.clusterUser)

			ctx := context.Background()
			info := mockServerInfo("/test.Service/Method")

			resp, err := interceptor(ctx, req, info, mockHandler)

			require.NoError(t, err)
			assert.Equal(t, "response", resp)

			// Should not panic with missing metadata
			require.Equal(t, 1, logs.Len())
			logEntry := logs.All()[0]
			assert.Equal(t, zapcore.InfoLevel, logEntry.Level)
			assert.Equal(t, "success", logEntry.ContextMap()["result"])
		})
	}
}

// TestAuditInterceptor_DurationMeasurement tests duration is correctly measured
func TestAuditInterceptor_DurationMeasurement(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditInterceptor(logger)

	// Mock handler that sleeps for 100ms
	slowHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "response", nil
	}

	req := createAuditTestMessage("cluster-123", "infobase-456", "admin")

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	start := time.Now()
	resp, err := interceptor(ctx, req, info, slowHandler)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Check logs
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	durationMs := logEntry.ContextMap()["duration_ms"].(int64)
	assert.GreaterOrEqual(t, durationMs, int64(100), "Duration should be at least 100ms")
	assert.LessOrEqual(t, durationMs, int64(elapsed.Milliseconds()+10), "Duration should match actual time")
}

// TestAuditInterceptor_ConcurrentAuditLogs tests concurrent operations don't interfere
func TestAuditInterceptor_ConcurrentAuditLogs(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditInterceptor(logger)

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			req := createAuditTestMessage(
				"cluster-"+string(rune(index)),
				"infobase-"+string(rune(index)),
				"user-"+string(rune(index)),
			)

			ctx := context.Background()
			info := mockServerInfo("/test.Service/Method")

			resp, err := interceptor(ctx, req, info, mockHandler)
			require.NoError(t, err)
			assert.Equal(t, "response", resp)
		}(i)
	}

	wg.Wait()

	// All operations should be logged
	assert.Equal(t, numGoroutines, logs.Len(), "All operations should have audit logs")

	// Check all logs have required fields
	for _, logEntry := range logs.All() {
		assert.Equal(t, zapcore.InfoLevel, logEntry.Level)
		assert.Equal(t, "success", logEntry.ContextMap()["result"])
		assert.Contains(t, logEntry.ContextMap(), "duration_ms")
	}
}

// TestAuditInterceptor_AllInfobaseMethods tests all 5 infobase methods are audited
func TestAuditInterceptor_AllInfobaseMethods(t *testing.T) {
	methods := []struct {
		name     string
		method   string
		expected zapcore.Level
	}{
		{"CreateInfobase", "/infobase.service.InfobaseManagementService/CreateInfobase", zapcore.InfoLevel},
		{"UpdateInfobase", "/infobase.service.InfobaseManagementService/UpdateInfobase", zapcore.InfoLevel},
		{"DropInfobase", "/infobase.service.InfobaseManagementService/DropInfobase", zapcore.WarnLevel},
		{"LockInfobase", "/infobase.service.InfobaseManagementService/LockInfobase", zapcore.InfoLevel},
		{"UnlockInfobase", "/infobase.service.InfobaseManagementService/UnlockInfobase", zapcore.InfoLevel},
	}

	for _, tt := range methods {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)
			interceptor := AuditInterceptor(logger)

			req := createAuditTestMessage("cluster-123", "infobase-456", "admin")

			ctx := context.Background()
			info := mockServerInfo(tt.method)

			resp, err := interceptor(ctx, req, info, mockHandler)

			require.NoError(t, err)
			assert.Equal(t, "response", resp)

			// Check log level
			require.Equal(t, 1, logs.Len())
			logEntry := logs.All()[0]

			assert.Equal(t, tt.expected, logEntry.Level, "%s should have correct log level", tt.name)
			assert.Equal(t, tt.method, logEntry.ContextMap()["operation"])
		})
	}
}

// TestAuditInterceptor_GenericError tests generic (non-gRPC) error logging
func TestAuditInterceptor_GenericError(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditInterceptor(logger)

	// Mock handler that returns generic error
	errorHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, errors.New("database connection failed")
	}

	req := createAuditTestMessage("cluster-123", "infobase-456", "admin")

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, errorHandler)

	require.Error(t, err)
	assert.Nil(t, resp)

	// Check logs
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	assert.Equal(t, zapcore.ErrorLevel, logEntry.Level)
	assert.Equal(t, "error", logEntry.ContextMap()["result"])
	// Generic errors don't have grpc_code
	assert.NotContains(t, logEntry.ContextMap(), "grpc_code")
}

// TestAuditInterceptor_NonProtoMessage tests audit with non-proto message
func TestAuditInterceptor_NonProtoMessage(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditInterceptor(logger)

	req := "not a proto message"

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Should still log, but without metadata
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	assert.Equal(t, zapcore.InfoLevel, logEntry.Level)
	assert.NotContains(t, logEntry.ContextMap(), "cluster_id")
	assert.NotContains(t, logEntry.ContextMap(), "infobase_id")
	assert.NotContains(t, logEntry.ContextMap(), "user")
}

// TestExtractAuditMetadata tests metadata extraction directly
func TestExtractAuditMetadata(t *testing.T) {
	tests := []struct {
		name              string
		clusterID         string
		infobaseID        string
		clusterUser       string
		expectedCluster   string
		expectedInfobase  string
		expectedUser      string
	}{
		{"all fields", "cluster-123", "infobase-456", "admin", "cluster-123", "infobase-456", "admin"},
		{"empty cluster", "", "infobase-456", "admin", "", "infobase-456", "admin"},
		{"empty infobase", "cluster-123", "", "admin", "cluster-123", "", "admin"},
		{"empty user", "cluster-123", "infobase-456", "", "cluster-123", "infobase-456", ""},
		{"all empty", "", "", "", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := createAuditTestMessage(tt.clusterID, tt.infobaseID, tt.clusterUser)
			metadata := extractAuditMetadata(msg)

			assert.Equal(t, tt.expectedCluster, metadata.ClusterID)
			assert.Equal(t, tt.expectedInfobase, metadata.InfobaseID)
			assert.Equal(t, tt.expectedUser, metadata.ClusterUser)
		})
	}
}

// TestAuditStreamInterceptor tests streaming interceptor
func TestAuditStreamInterceptor(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditStreamInterceptor(logger)

	mockStreamHandler := func(srv interface{}, stream grpc.ServerStream) error {
		time.Sleep(50 * time.Millisecond) // Simulate some work
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.Service/StreamMethod",
		IsClientStream: true,
		IsServerStream: true,
	}

	err := interceptor(nil, nil, info, mockStreamHandler)
	assert.NoError(t, err)

	// Check logs
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	assert.Equal(t, zapcore.InfoLevel, logEntry.Level)
	assert.Equal(t, "gRPC stream completed", logEntry.Message)
	assert.Equal(t, "/test.Service/StreamMethod", logEntry.ContextMap()["operation"])
	assert.True(t, logEntry.ContextMap()["is_client_stream"].(bool))
	assert.True(t, logEntry.ContextMap()["is_server_stream"].(bool))

	// Check duration
	durationMs := logEntry.ContextMap()["duration_ms"].(int64)
	assert.GreaterOrEqual(t, durationMs, int64(50))
}

// TestAuditStreamInterceptor_Error tests streaming interceptor with error
func TestAuditStreamInterceptor_Error(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditStreamInterceptor(logger)

	mockStreamHandler := func(srv interface{}, stream grpc.ServerStream) error {
		return status.Error(codes.Internal, "stream processing failed")
	}

	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.Service/StreamMethod",
		IsClientStream: true,
		IsServerStream: false,
	}

	err := interceptor(nil, nil, info, mockStreamHandler)
	require.Error(t, err)

	// Check logs
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	assert.Equal(t, zapcore.InfoLevel, logEntry.Level)
	assert.Equal(t, "error", logEntry.ContextMap()["result"])
}

// TestResultStatus_AllCases tests resultStatus function with all cases
func TestResultStatus_AllCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"no error", nil, "success"},
		{"gRPC OK", status.Error(codes.OK, "ok"), "success"},
		{"gRPC NotFound", status.Error(codes.NotFound, "not found"), "error"},
		{"gRPC Internal", status.Error(codes.Internal, "internal"), "error"},
		{"gRPC Unavailable", status.Error(codes.Unavailable, "unavailable"), "error"},
		{"generic error", errors.New("generic"), "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resultStatus(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAuditInterceptor_SpecialCharactersInMetadata tests that special characters in metadata are handled correctly
func TestAuditInterceptor_SpecialCharactersInMetadata(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	interceptor := AuditInterceptor(logger)

	// Try to inject JSON in user field
	maliciousUser := `admin", "injected": "malicious`
	req := createAuditTestMessage("cluster-123", "infobase-456", maliciousUser)

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Check logs
	require.Equal(t, 1, logs.Len())
	logEntry := logs.All()[0]

	// zap stores the field value as-is (escaping happens during JSON marshaling)
	userField := logEntry.ContextMap()["user"].(string)
	assert.Equal(t, maliciousUser, userField, "User field should be stored as-is, escaping happens at marshaling time")

	// Verify the log entry is valid (no panic, no corruption)
	assert.Equal(t, "success", logEntry.ContextMap()["result"])
}
