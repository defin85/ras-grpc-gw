package server

import (
	"context"
	"testing"

	clientv1 "github.com/v8platform/protos/gen/ras/client/v1"
	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

// TestCreateInfobase_WithMock demonstrates how to test gRPC methods using MockRASClient
// This example shows the benefit of dependency injection for testing
func TestCreateInfobase_WithMock(t *testing.T) {
	// Setup mock RAS client
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			// Simulate error from RAS server
			return nil, status.Error(codes.Unavailable, "RAS server unavailable")
		},
	}

	// Create server with mocked client (instead of real RAS connection)
	server := NewInfobaseManagementServer(mockClient)

	// Test CreateInfobase when RAS is unavailable
	req := &pb.CreateInfobaseRequest{
		ClusterId: "test-cluster",
		Name:      "test-db",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb",
	}

	resp, err := server.CreateInfobase(context.Background(), req)

	// Verify error handling
	if err == nil {
		t.Fatal("Expected error when RAS is unavailable, got nil")
	}

	if resp != nil {
		t.Errorf("Expected nil response on error, got %v", resp)
	}

	// Verify error type
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.Unavailable {
		t.Errorf("Expected Unavailable error, got %v", st.Code())
	}
}

// TestUpdateInfobase_WithSuccessfulMock demonstrates successful mock scenario
func TestUpdateInfobase_WithSuccessfulMock(t *testing.T) {
	// Setup successful mock
	mockEndpoint := &MockEndpoint{
		RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
			// Simulate successful response
			return req.Respond, nil
		},
	}

	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return mockEndpoint, nil
		},
	}

	server := NewInfobaseManagementServer(mockClient)

	// Test UpdateInfobase with successful mock
	sessionsDeny := true
	req := &pb.UpdateInfobaseRequest{
		ClusterId:    "test-cluster",
		InfobaseId:   "test-infobase",
		SessionsDeny: &sessionsDeny,
	}

	resp, err := server.UpdateInfobase(context.Background(), req)

	// Verify success
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.InfobaseId != req.InfobaseId {
		t.Errorf("Expected infobase_id=%s, got %s", req.InfobaseId, resp.InfobaseId)
	}
}

// TestRASClient_Interface ensures adapter implements interface correctly
func TestRASClient_Interface(t *testing.T) {
	// Verify that NewRASClient returns RASClient interface
	var _ RASClient = NewRASClient("localhost:1545")

	// Verify that MockRASClient implements interface
	var _ RASClient = &MockRASClient{}
}
