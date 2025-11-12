// Package interceptor provides gRPC interceptors for security and audit logging.
//
// # Overview
//
// This package contains production-ready interceptors for gRPC servers:
//   - Password Sanitization: Automatically redacts password fields in logs
//   - Audit Logging: Structured JSON logging of all gRPC operations
//
// # Password Sanitization
//
// The password sanitization interceptor uses protobuf reflection to automatically
// detect and redact password fields in gRPC requests before logging them.
// It identifies password fields by the naming convention "*_password" (e.g.,
// cluster_password, db_password, infobase_password).
//
// The interceptor creates a clone of the request message for logging purposes
// and replaces password values with "******", while the original request passed
// to the handler remains unchanged.
//
// Example usage:
//
//	import "github.com/khorevaa/ras-grpc-gw/pkg/interceptor"
//
//	logger := zap.NewProduction()
//	server := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(
//	        interceptor.SanitizePasswordsInterceptor(logger),
//	        interceptor.AuditInterceptor(logger),
//	    ),
//	)
//
// # Audit Logging
//
// The audit logging interceptor records all gRPC operations in structured JSON format
// using go.uber.org/zap. It automatically extracts metadata from requests
// (cluster_id, infobase_id, user) and measures operation duration.
//
// Log levels:
//   - INFO: Successful operations
//   - WARN: Destructive operations (e.g., DropInfobase)
//   - ERROR: Failed operations with gRPC error codes
//
// Example audit log entry:
//
//	{
//	  "level": "info",
//	  "timestamp": "2025-11-02T22:00:00.000Z",
//	  "operation": "/infobase.service.InfobaseManagementService/CreateInfobase",
//	  "cluster_id": "uuid-123",
//	  "infobase_id": "uuid-456",
//	  "user": "admin",
//	  "result": "success",
//	  "duration_ms": 1234
//	}
//
// # Interceptor Chain Order
//
// IMPORTANT: The order of interceptors matters for security!
//
// Always place SanitizePasswordsInterceptor BEFORE AuditInterceptor:
//
//	grpc.ChainUnaryInterceptor(
//	    interceptor.SanitizePasswordsInterceptor(logger),  // 1. Sanitize first
//	    interceptor.AuditInterceptor(logger),              // 2. Audit second
//	)
//
// This ensures that the audit log sees sanitized passwords, while the handler
// receives the original request with actual passwords.
//
// # Performance
//
// Both interceptors are optimized for production use:
//   - Password sanitization: <1ms overhead per request
//   - Audit logging: <500µs overhead per request
//   - No memory leaks or goroutine leaks
//   - Concurrent-safe (tested with 100+ concurrent requests)
//
// # Thread Safety
//
// All interceptors are thread-safe and can be used in highly concurrent environments.
// The password sanitization uses proto.Clone() which creates a deep copy,
// ensuring no shared state between goroutines.
package interceptor
