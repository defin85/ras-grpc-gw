package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/v8platform/ras-grpc-gw/pkg/logger"
	"go.uber.org/zap"
)

// TerminateSessionHTTPRequest represents HTTP request for terminating a session
type TerminateSessionHTTPRequest struct {
	ClusterID string `json:"cluster_id"`
	SessionID string `json:"session_id"`
}

// TerminateSessionHTTPResponse represents HTTP response
type TerminateSessionHTTPResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// HandleTerminateSession обрабатывает HTTP запрос на завершение сессии
// POST /api/v1/sessions/terminate
// Body: {"cluster_id": "uuid", "session_id": "uuid"}
func (s *rasClientServiceServer) HandleTerminateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req TerminateSessionHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Failed to decode TerminateSession HTTP request", zap.Error(err))
		respondJSON(w, http.StatusBadRequest, TerminateSessionHTTPResponse{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	// Validate params
	if req.ClusterID == "" || req.SessionID == "" {
		respondJSON(w, http.StatusBadRequest, TerminateSessionHTTPResponse{
			Success: false,
			Error:   "cluster_id and session_id are required",
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call gRPC method
	terminateReq := &TerminateSessionRequest{
		ClusterId: req.ClusterID,
		SessionId: req.SessionID,
	}

	_, err := s.TerminateSession(ctx, terminateReq)
	if err != nil {
		logger.Log.Error("TerminateSession HTTP handler failed",
			zap.String("cluster_id", req.ClusterID),
			zap.String("session_id", req.SessionID),
			zap.Error(err))

		respondJSON(w, http.StatusInternalServerError, TerminateSessionHTTPResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Success
	respondJSON(w, http.StatusOK, TerminateSessionHTTPResponse{
		Success: true,
		Message: "Session terminated successfully",
	})
}

// respondJSON writes JSON response
func respondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}
