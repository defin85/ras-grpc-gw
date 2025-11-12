package server

// TerminateSessionRequest represents a request to terminate a session in the 1C cluster
// This message is not part of the official v8platform/protos v0.2.0 API
// and is implemented based on reverse-engineered RAS protocol (message type 0x47)
type TerminateSessionRequest struct {
	// ClusterId is the UUID of the 1C cluster
	ClusterId string `json:"cluster_id"`

	// SessionId is the UUID of the session to terminate
	SessionId string `json:"session_id"`
}
