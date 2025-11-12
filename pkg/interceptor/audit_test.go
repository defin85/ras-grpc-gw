package interceptor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func createTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	return zap.New(core), logs
}

func TestAuditInterceptor_BasicTest(t *testing.T) {
	logger, logs := createTestLogger()
	interceptor := AuditInterceptor(logger)

	req := "test-request"

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)
	require.Equal(t, 1, logs.Len())
}

func TestResultStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"no error", nil, "success"},
		{"gRPC error", status.Error(codes.NotFound, "not found"), "error"},
		{"generic error", errors.New("generic error"), "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resultStatus(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
