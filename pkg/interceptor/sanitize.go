package interceptor

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	passwordMask     = "******"
	passwordSuffix   = "_password"
)

// SanitizePasswordsInterceptor automatically sanitizes password fields in gRPC requests
// before logging them. It uses protobuf reflection to find all fields ending with "*_password"
// and replaces their values with "******" for non-empty passwords.
//
// This interceptor does NOT modify the actual request passed to the handler - it only
// sanitizes the logged version for security purposes.
func SanitizePasswordsInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Sanitize and log the request
		if protoMsg, ok := req.(proto.Message); ok {
			sanitized := sanitizeMessage(protoMsg)
			logger.Debug("received gRPC request",
				zap.String("method", info.FullMethod),
				zap.Any("request", sanitized),
			)
		} else {
			// Fallback if not a proto message
			logger.Debug("received gRPC request",
				zap.String("method", info.FullMethod),
				zap.String("warning", "request is not a proto.Message"),
			)
		}

		// Call the actual handler with the ORIGINAL request
		return handler(ctx, req)
	}
}

// SanitizePasswordsStreamInterceptor is the streaming version of SanitizePasswordsInterceptor.
func SanitizePasswordsStreamInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		logger.Debug("received gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
		)

		// Call the actual handler
		return handler(srv, ss)
	}
}

// sanitizeMessage creates a deep copy of the proto message and replaces all password fields
// with "******" for non-empty values, or keeps empty string for empty passwords.
func sanitizeMessage(msg proto.Message) proto.Message {
	// Create a deep clone to avoid modifying the original
	clone := proto.Clone(msg)

	// Use protobuf reflection to iterate over fields
	reflectMsg := clone.ProtoReflect()
	reflectMsg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldName := string(fd.Name())

		// Check if field name ends with "_password"
		if strings.HasSuffix(fieldName, passwordSuffix) {
			// Only sanitize non-empty string values
			if fd.Kind() == protoreflect.StringKind {
				currentValue := v.String()
				if currentValue != "" {
					reflectMsg.Set(fd, protoreflect.ValueOfString(passwordMask))
				}
				// Empty passwords remain empty (don't mask them)
			}
		}

		return true // Continue iteration
	})

	return clone
}

// SanitizePasswordInString is a helper function to sanitize passwords in plain strings.
// Useful for logging non-protobuf data.
func SanitizePasswordInString(s string) string {
	if s == "" {
		return ""
	}
	return passwordMask
}
