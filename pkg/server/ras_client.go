package server

import (
	"context"

	clientv1 "github.com/v8platform/protos/gen/ras/client/v1"
	"github.com/v8platform/ras-grpc-gw/pkg/client"
)

// RASClient is an interface for interacting with RAS server
// This abstraction enables dependency injection and mocking in tests
type RASClient interface {
	// GetEndpoint returns an endpoint for communicating with RAS server
	// The endpoint maintains authentication state and is used for all RAS operations
	GetEndpoint(ctx context.Context) (clientv1.EndpointServiceImpl, error)
}

// clientConnAdapter adapts client.ClientConn to RASClient interface
// This allows existing code to work with the new interface without changes
type clientConnAdapter struct {
	conn *client.ClientConn
}

// NewRASClient creates a new RASClient from ClientConn
func NewRASClient(rasAddr string) RASClient {
	return &clientConnAdapter{
		conn: client.NewClientConn(rasAddr),
	}
}

// GetEndpoint implements RASClient interface
func (a *clientConnAdapter) GetEndpoint(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
	return a.conn.GetEndpoint(ctx)
}
