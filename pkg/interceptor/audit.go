package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Destructive operations that require warning-level logging
var destructiveOperations = map[string]bool{
	"/infobase.service.InfobaseManagementService/DropInfobase": true,
}

// AuditInterceptor logs all gRPC operations with structured metadata in JSON format.
// It captures operation details, user information, execution time, and result status.
//
// Log levels:
// - WARN: Destructive operations (DropInfobase)
// - ERROR: Failed operations
// - INFO: Successful operations
func AuditInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Extract metadata from request
		var metadata auditMetadata
		if protoMsg, ok := req.(proto.Message); ok {
			metadata = extractAuditMetadata(protoMsg)
		}

		// Call the actual handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Log audit entry
		logAuditEntry(logger, info.FullMethod, metadata, err, duration)

		return resp, err
	}
}

// AuditStreamInterceptor is the streaming version of AuditInterceptor.
func AuditStreamInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call the actual handler
		err := handler(srv, ss)

		// Calculate duration
		duration := time.Since(start)

		// Log audit entry for stream
		logger.Info("gRPC stream completed",
			zap.String("operation", info.FullMethod),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.String("result", resultStatus(err)),
			zap.Error(err),
		)

		return err
	}
}

// auditMetadata holds extracted metadata from gRPC requests
type auditMetadata struct {
	ClusterID   string
	InfobaseID  string
	ClusterUser string
}

// extractAuditMetadata extracts relevant metadata from proto message using reflection
func extractAuditMetadata(msg proto.Message) auditMetadata {
	metadata := auditMetadata{}

	reflectMsg := msg.ProtoReflect()
	reflectMsg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldName := string(fd.Name())

		// Extract cluster_id
		if fieldName == "cluster_id" && fd.Kind() == protoreflect.StringKind {
			metadata.ClusterID = v.String()
		}

		// Extract infobase_id
		if fieldName == "infobase_id" && fd.Kind() == protoreflect.StringKind {
			metadata.InfobaseID = v.String()
		}

		// Extract cluster_user
		if fieldName == "cluster_user" && fd.Kind() == protoreflect.StringKind {
			metadata.ClusterUser = v.String()
		}

		return true
	})

	return metadata
}

// logAuditEntry writes a structured audit log entry
func logAuditEntry(logger *zap.Logger, fullMethod string, metadata auditMetadata, err error, duration time.Duration) {
	// Base fields
	fields := []zap.Field{
		zap.String("operation", fullMethod),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.String("result", resultStatus(err)),
	}

	// Add metadata if available
	if metadata.ClusterID != "" {
		fields = append(fields, zap.String("cluster_id", metadata.ClusterID))
	}
	if metadata.InfobaseID != "" {
		fields = append(fields, zap.String("infobase_id", metadata.InfobaseID))
	}
	if metadata.ClusterUser != "" {
		fields = append(fields, zap.String("user", metadata.ClusterUser))
	}

	// Add error if present
	if err != nil {
		fields = append(fields, zap.Error(err))
		
		// Add gRPC status code
		if st, ok := status.FromError(err); ok {
			fields = append(fields, zap.String("grpc_code", st.Code().String()))
		}
	}

	// Determine log level
	if err != nil {
		logger.Error("gRPC operation failed", fields...)
	} else if destructiveOperations[fullMethod] {
		logger.Warn("gRPC destructive operation", fields...)
	} else {
		logger.Info("gRPC operation completed", fields...)
	}
}

// resultStatus converts error to human-readable status
func resultStatus(err error) string {
	if err != nil {
		// Try to get gRPC status code
		if st, ok := status.FromError(err); ok {
			code := st.Code()
			if code == codes.OK {
				return "success"
			}
			return "error"
		}
		return "error"
	}
	return "success"
}
