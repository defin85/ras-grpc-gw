package server

import (
	"context"

	clientv1 "github.com/v8platform/protos/gen/ras/client/v1"
	"google.golang.org/protobuf/types/known/anypb"
)

// MockRASClient is a mock implementation of RASClient for testing
type MockRASClient struct {
	GetEndpointFunc func(ctx context.Context) (clientv1.EndpointServiceImpl, error)
}

func (m *MockRASClient) GetEndpoint(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
	if m.GetEndpointFunc != nil {
		return m.GetEndpointFunc(ctx)
	}
	return nil, nil
}

// MockEndpoint is a mock implementation of EndpointServiceImpl for testing
type MockEndpoint struct {
	RequestFunc func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error)
}

func (m *MockEndpoint) Request(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
	if m.RequestFunc != nil {
		return m.RequestFunc(ctx, req)
	}
	return nil, nil
}

// Implement other methods from EndpointServiceImpl if needed
// (For now, only Request is critical for our tests)
