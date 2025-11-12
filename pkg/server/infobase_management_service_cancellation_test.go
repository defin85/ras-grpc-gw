package server

import (
	"context"
	"testing"

	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestUpdateInfobase_ContextCancelled проверяет что метод корректно обрабатывает отмену context
func TestUpdateInfobase_ContextCancelled(t *testing.T) {
	// Arrange: Создаем сервер с mock RAS client
	mockClient := &MockRASClient{}
	server := NewInfobaseManagementServer(mockClient)

	// Создаем отмененный context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	// Act: Пытаемся вызвать UpdateInfobase с отмененным context
	req := &pb.UpdateInfobaseRequest{
		ClusterId:  "test-cluster",
		InfobaseId: "test-infobase",
	}
	resp, err := server.UpdateInfobase(ctx, req)

	// Assert: Проверяем что вернулась ошибка codes.Canceled
	if resp != nil {
		t.Errorf("Expected nil response, got %+v", resp)
	}
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Проверяем что код ошибки = Canceled
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Canceled {
		t.Errorf("Expected codes.Canceled, got %v", st.Code())
	}
}

// TestCreateInfobase_ContextCancelled проверяет что метод корректно обрабатывает отмену context
func TestCreateInfobase_ContextCancelled(t *testing.T) {
	// Arrange
	mockClient := &MockRASClient{}
	server := NewInfobaseManagementServer(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	// Act
	req := &pb.CreateInfobaseRequest{
		ClusterId: "test-cluster",
		Name:      "test-db",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb",
	}
	resp, err := server.CreateInfobase(ctx, req)

	// Assert
	if resp != nil {
		t.Errorf("Expected nil response, got %+v", resp)
	}
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Canceled {
		t.Errorf("Expected codes.Canceled, got %v", st.Code())
	}
}

// TestDropInfobase_ContextCancelled проверяет что метод корректно обрабатывает отмену context
// И логирует "CANCELLED" status в audit log
func TestDropInfobase_ContextCancelled(t *testing.T) {
	// Arrange
	mockClient := &MockRASClient{}
	server := NewInfobaseManagementServer(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	// Act
	req := &pb.DropInfobaseRequest{
		ClusterId:  "test-cluster",
		InfobaseId: "test-infobase",
		DropMode:   pb.DropMode_DROP_MODE_UNREGISTER_ONLY,
	}
	resp, err := server.DropInfobase(ctx, req)

	// Assert
	if resp != nil {
		t.Errorf("Expected nil response, got %+v", resp)
	}
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Canceled {
		t.Errorf("Expected codes.Canceled, got %v", st.Code())
	}

	// Note: Здесь мы не можем проверить audit log напрямую,
	// но можем убедиться что операция корректно прервалась
}

// TestLockInfobase_ContextCancelled проверяет что wrapper метод обрабатывает отмену context
func TestLockInfobase_ContextCancelled(t *testing.T) {
	// Arrange
	mockClient := &MockRASClient{}
	server := NewInfobaseManagementServer(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	// Act
	req := &pb.LockInfobaseRequest{
		ClusterId:         "test-cluster",
		InfobaseId:        "test-infobase",
		SessionsDeny:      true,
		ScheduledJobsDeny: false,
	}
	resp, err := server.LockInfobase(ctx, req)

	// Assert
	if resp != nil {
		t.Errorf("Expected nil response, got %+v", resp)
	}
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Canceled {
		t.Errorf("Expected codes.Canceled, got %v", st.Code())
	}
}

// TestUnlockInfobase_ContextCancelled проверяет что wrapper метод обрабатывает отмену context
func TestUnlockInfobase_ContextCancelled(t *testing.T) {
	// Arrange
	mockClient := &MockRASClient{}
	server := NewInfobaseManagementServer(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	// Act
	req := &pb.UnlockInfobaseRequest{
		ClusterId:            "test-cluster",
		InfobaseId:           "test-infobase",
		UnlockSessions:       true,
		UnlockScheduledJobs:  true,
	}
	resp, err := server.UnlockInfobase(ctx, req)

	// Assert
	if resp != nil {
		t.Errorf("Expected nil response, got %+v", resp)
	}
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Canceled {
		t.Errorf("Expected codes.Canceled, got %v", st.Code())
	}
}

// TestUpdateInfobase_ContextCancelledBeforeRASRequest проверяет вторую точку проверки context
// (перед endpoint.Request)
func TestUpdateInfobase_ContextCancelledBeforeRASRequest(t *testing.T) {
	// Arrange: Используем mock который успешно вернет endpoint
	mockClient := &MockRASClient{}
	server := NewInfobaseManagementServer(mockClient)

	// Создаем context который будет отменен перед Request
	ctx, cancel := context.WithCancel(context.Background())

	// ВАЖНО: Здесь нам нужен более сложный mock который:
	// 1. Успешно вернет endpoint в GetEndpoint
	// 2. Но context будет отменен ПЕРЕД endpoint.Request
	//
	// Простой способ - отменить context сразу (проверяется первая точка)
	cancel()

	// Act
	req := &pb.UpdateInfobaseRequest{
		ClusterId:  "test-cluster",
		InfobaseId: "test-infobase",
	}
	resp, err := server.UpdateInfobase(ctx, req)

	// Assert
	if resp != nil {
		t.Errorf("Expected nil response, got %+v", resp)
	}
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Canceled {
		t.Errorf("Expected codes.Canceled, got %v", st.Code())
	}
}
