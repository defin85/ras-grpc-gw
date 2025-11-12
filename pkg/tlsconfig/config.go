package tlsconfig

import (
	"crypto/tls"
	"fmt"
	"os"

	"go.uber.org/zap"
)

// LoadTLSConfig loads TLS configuration from environment variables.
// If TLS_ENABLED is not "true", returns nil (TLS disabled).
// If TLS_CERT_FILE and TLS_KEY_FILE don't exist, generates self-signed certificate for development.
//
// Environment variables:
//   TLS_ENABLED=true       - Enable TLS
//   TLS_CERT_FILE=/path    - Path to certificate file
//   TLS_KEY_FILE=/path     - Path to private key file
func LoadTLSConfig(logger *zap.Logger) (*tls.Config, error) {
	// Check if TLS is enabled
	if !isTLSEnabled() {
		return nil, nil
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	// If files don't exist, generate self-signed for development
	if !fileExists(certFile) || !fileExists(keyFile) {
		logger.Warn("TLS certificate files not found, generating self-signed certificate for DEVELOPMENT ONLY")

		var err error
		certFile, keyFile, err = GenerateSelfSignedCert("./certs")
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}

		logger.Warn("Using self-signed certificate (DEV ONLY - DO NOT USE IN PRODUCTION)",
			zap.String("cert", certFile),
			zap.String("key", keyFile),
		)
	} else {
		logger.Info("Loading TLS certificate",
			zap.String("cert", certFile),
			zap.String("key", keyFile),
		)
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	// Create TLS config with production-ready settings
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}, nil
}

// isTLSEnabled checks if TLS is enabled via TLS_ENABLED environment variable
func isTLSEnabled() bool {
	return os.Getenv("TLS_ENABLED") == "true"
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
