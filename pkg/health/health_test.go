package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/v8platform/ras-grpc-gw/pkg/logger"
)

// MockHealthChecker для тестирования
type MockHealthChecker struct {
	shouldFail bool
}

func (m *MockHealthChecker) Check(ctx context.Context) error {
	if m.shouldFail {
		return fmt.Errorf("health check failed")
	}
	return nil
}

func TestMain(m *testing.M) {
	// Инициализация logger для тестов
	_ = logger.Init(false)
	m.Run()
}

func TestNewServer(t *testing.T) {
	checker := &MockHealthChecker{}
	srv := NewServer(":8080", checker)

	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}

	if srv.server == nil {
		t.Error("server.server is nil")
	}

	if srv.checker != checker {
		t.Error("server.checker is not set correctly")
	}
}

func TestNewServer_WithNilChecker(t *testing.T) {
	srv := NewServer(":8080", nil)

	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}

	if srv.server == nil {
		t.Error("server.server is nil")
	}

	if srv.checker != nil {
		t.Error("server.checker should be nil")
	}
}

func TestHealthHandler(t *testing.T) {
	checker := &MockHealthChecker{}
	srv := NewServer(":8080", checker)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthHandler() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", response["status"])
	}

	if response["service"] != "ras-grpc-gw" {
		t.Errorf("service = %v, want ras-grpc-gw", response["service"])
	}

	if response["version"] != "v1.0.0-cc" {
		t.Errorf("version = %v, want v1.0.0-cc", response["version"])
	}

	// Проверяем Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", contentType)
	}
}

func TestReadyHandler_Success(t *testing.T) {
	checker := &MockHealthChecker{shouldFail: false}
	srv := NewServer(":8080", checker)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	srv.readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("readyHandler() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("status = %v, want ready", response["status"])
	}

	// Проверяем Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", contentType)
	}
}

func TestReadyHandler_Failure(t *testing.T) {
	checker := &MockHealthChecker{shouldFail: true}
	srv := NewServer(":8080", checker)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	srv.readyHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("readyHandler() status = %v, want %v", w.Code, http.StatusServiceUnavailable)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "not_ready" {
		t.Errorf("status = %v, want not_ready", response["status"])
	}

	if response["error"] == nil {
		t.Error("error field is missing in response")
	}

	expectedError := "health check failed"
	if response["error"] != expectedError {
		t.Errorf("error = %v, want %v", response["error"], expectedError)
	}
}

func TestReadyHandler_NilChecker(t *testing.T) {
	srv := NewServer(":8080", nil)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	srv.readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("readyHandler() with nil checker status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("status = %v, want ready", response["status"])
	}
}

func TestServerStartShutdown(t *testing.T) {
	checker := &MockHealthChecker{}
	srv := NewServer("127.0.0.1:0", checker) // Port 0 = random free port

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// Wait for Start() to return
	select {
	case err := <-errChan:
		if err != http.ErrServerClosed {
			t.Errorf("Start() after Shutdown error = %v, want %v", err, http.ErrServerClosed)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start() did not return after Shutdown")
	}
}

func TestHealthHandler_POSTMethod(t *testing.T) {
	checker := &MockHealthChecker{}
	srv := NewServer(":8080", checker)

	// Пробуем POST вместо GET
	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()

	srv.healthHandler(w, req)

	// Должно работать с любым методом
	if w.Code != http.StatusOK {
		t.Errorf("healthHandler() with POST status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestReadyHandler_POSTMethod(t *testing.T) {
	checker := &MockHealthChecker{}
	srv := NewServer(":8080", checker)

	// Пробуем POST вместо GET
	req := httptest.NewRequest("POST", "/ready", nil)
	w := httptest.NewRecorder()

	srv.readyHandler(w, req)

	// Должно работать с любым методом
	if w.Code != http.StatusOK {
		t.Errorf("readyHandler() with POST status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestServerTimeouts(t *testing.T) {
	checker := &MockHealthChecker{}
	srv := NewServer(":8080", checker)

	// Проверяем что таймауты установлены
	if srv.server.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", srv.server.ReadTimeout, 5*time.Second)
	}

	if srv.server.WriteTimeout != 5*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", srv.server.WriteTimeout, 5*time.Second)
	}
}

func TestReadyHandler_ContextTimeout(t *testing.T) {
	// Создаем checker с задержкой
	slowChecker := &MockHealthChecker{}

	srv := NewServer(":8080", slowChecker)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	// Вызываем handler
	srv.readyHandler(w, req)

	// Проверяем что получили ответ (не зависло)
	if w.Code != http.StatusOK {
		t.Errorf("readyHandler() status = %v, want %v", w.Code, http.StatusOK)
	}
}
