package server

import (
	"context"
	"testing"

	ras_service "github.com/v8platform/protos/gen/ras/service/api/v1"
	"github.com/v8platform/ras-grpc-gw/pkg/logger"
	"google.golang.org/grpc"
	"time"
)

func TestMain(m *testing.M) {
	// Инициализация logger для тестов
	_ = logger.Init(false)
	m.Run()
}


func TestNewRASServer(t *testing.T) {
	rasAddr := "localhost:1545"
	srv := NewRASServer(rasAddr)

	if srv == nil {
		t.Fatal("NewRASServer() returned nil")
	}

	// Проверяем что адрес установлен (приватное поле, но можем проверить через Check)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := srv.Check(ctx)
	if err != nil {
		t.Errorf("Check() error = %v, expected nil for configured server", err)
	}
}

func TestRASServer_Check_EmptyAddress(t *testing.T) {
	srv := &RASServer{
		rasAddr: "",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := srv.Check(ctx)
	if err == nil {
		t.Error("Check() with empty address should return error")
	}

	expectedError := "RAS address not configured"
	if err.Error() != expectedError {
		t.Errorf("Check() error = %v, want %v", err, expectedError)
	}
}

func TestRASServer_Check_WithContext(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := srv.Check(ctx)
	if err != nil {
		t.Errorf("Check() error = %v", err)
	}
}

func TestRASServer_Check_CanceledContext(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := srv.Check(ctx)
	if err != context.Canceled {
		t.Errorf("Check() with canceled context error = %v, want %v", err, context.Canceled)
	}
}

func TestRASServer_Check_DeadlineExceeded(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	err := srv.Check(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Check() with expired deadline error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestRASServer_GracefulStop_WithoutStart(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Calling GracefulStop without starting should not panic
	err := srv.GracefulStop(ctx)
	if err != nil {
		t.Errorf("GracefulStop() without start error = %v", err)
	}
}

func TestRASServer_GracefulStop_NilServer(t *testing.T) {
	srv := &RASServer{
		rasAddr:    "localhost:1545",
		grpcServer: nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// GracefulStop с nil grpcServer не должен вызывать панику
	err := srv.GracefulStop(ctx)
	if err != nil {
		t.Errorf("GracefulStop() with nil grpcServer error = %v", err)
	}
}

func TestNewRASServer_WithDifferentAddresses(t *testing.T) {
	tests := []struct {
		name    string
		rasAddr string
	}{
		{
			name:    "localhost with port",
			rasAddr: "localhost:1545",
		},
		{
			name:    "IP address with port",
			rasAddr: "127.0.0.1:1545",
		},
		{
			name:    "hostname with port",
			rasAddr: "ras-server:1545",
		},
		{
			name:    "IP with different port",
			rasAddr: "192.168.1.100:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewRASServer(tt.rasAddr)
			if srv == nil {
				t.Fatal("NewRASServer() returned nil")
			}

			// Проверяем что сервер создан с правильным адресом
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := srv.Check(ctx)
			if err != nil {
				t.Errorf("Check() error = %v", err)
			}
		})
	}
}

func TestRASServer_Check_ValidAddress(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	ctx := context.Background()

	err := srv.Check(ctx)
	if err != nil {
		t.Errorf("Check() with valid address should not return error, got: %v", err)
	}
}

func TestRASServer_MultipleChecks(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	ctx := context.Background()

	// Выполняем несколько проверок подряд
	for i := 0; i < 5; i++ {
		err := srv.Check(ctx)
		if err != nil {
			t.Errorf("Check() iteration %d error = %v", i, err)
		}
	}
}

func TestRASServer_CheckWithTimeout(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	// Короткий таймаут
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Даем контексту истечь
	time.Sleep(20 * time.Millisecond)

	err := srv.Check(ctx)
	if err == nil {
		t.Error("Check() with expired context should return error")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Check() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestNewRasClientServiceServer(t *testing.T) {
	rasAddr := "localhost:1545"
	srv := NewRasClientServiceServer(rasAddr)

	if srv == nil {
		t.Fatal("NewRasClientServiceServer() returned nil")
	}

	// Проверяем что сервер реализует интерфейс
	_, ok := srv.(ras_service.RASServiceServer)
	if !ok {
		t.Error("NewRasClientServiceServer() does not implement RASServiceServer interface")
	}
}

func TestRASServer_EmptyStructInitialization(t *testing.T) {
	srv := &RASServer{}

	if srv.rasAddr != "" {
		t.Errorf("Empty RASServer should have empty rasAddr, got: %v", srv.rasAddr)
	}

	if srv.grpcServer != nil {
		t.Error("Empty RASServer should have nil grpcServer")
	}

	if srv.idxClients != nil {
		t.Error("Empty RASServer should have nil idxClients")
	}

	if srv.idxEndpoints != nil {
		t.Error("Empty RASServer should have nil idxEndpoints")
	}
}

func TestRASServer_Check_BackgroundContext(t *testing.T) {
	srv := NewRASServer("localhost:1545")

	// Используем context.Background() без таймаута
	err := srv.Check(context.Background())
	if err != nil {
		t.Errorf("Check() with background context error = %v", err)
	}
}

func TestNewAccessServer(t *testing.T) {
	srv := NewAccessServer()

	if srv == nil {
		t.Fatal("NewAccessServer() returned nil")
	}

	// Проверяем что сервер реализует интерфейс AccessServer
	_, ok := srv.(AccessServer)
	if !ok {
		t.Error("NewAccessServer() does not implement AccessServer interface")
	}
}

func TestRASServer_GracefulStop_WithTimeout(t *testing.T) {
	srv := &RASServer{
		rasAddr:    "localhost:1545",
		grpcServer: grpc.NewServer(),
	}

	// Создаем контекст с очень коротким таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Даем контексту истечь
	time.Sleep(10 * time.Millisecond)

	err := srv.GracefulStop(ctx)
	if err == nil {
		t.Error("GracefulStop() with expired context should return error")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("GracefulStop() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
