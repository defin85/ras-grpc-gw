package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/v8platform/ras-grpc-gw/pkg/logger"
	"go.uber.org/zap"
)

// HealthChecker interface для проверки здоровья компонентов
type HealthChecker interface {
	Check(ctx context.Context) error
}

// Server HTTP сервер для health checks
type Server struct {
	server  *http.Server
	checker HealthChecker
}

// NewServer создает новый health check сервер
func NewServer(addr string, checker HealthChecker) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		checker: checker,
	}

	// Регистрация endpoints
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)

	return s
}

// SetHandler добавляет кастомный handler к mux (для расширения API)
func (s *Server) SetHandler(pattern string, handler http.HandlerFunc) {
	if mux, ok := s.server.Handler.(*http.ServeMux); ok {
		mux.HandleFunc(pattern, handler)
	}
}

// Start запускает health check сервер
func (s *Server) Start() error {
	logger.Log.Info("Starting health check server", zap.String("address", s.server.Addr))
	return s.server.ListenAndServe()
}

// Shutdown gracefully останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Shutting down health check server")
	return s.server.Shutdown(ctx)
}

// healthHandler обрабатывает /health endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Health check всегда возвращает 200 если сервис запущен
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "ras-grpc-gw",
		"version": "v1.0.0-cc",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// readyHandler обрабатывает /ready endpoint
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Проверка готовности через checker
	if s.checker != nil {
		if err := s.checker.Check(ctx); err != nil {
			logger.Log.Warn("Readiness check failed", zap.Error(err))

			response := map[string]interface{}{
				"status": "not_ready",
				"error":  err.Error(),
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Готов к обработке запросов
	response := map[string]interface{}{
		"status": "ready",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
