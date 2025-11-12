package tlsconfig

import (
	"crypto/tls"
	"os"
	"testing"

	"go.uber.org/zap/zaptest"
)

// BenchmarkLoadTLSConfig measures TLS config loading overhead
func BenchmarkLoadTLSConfig(b *testing.B) {
	logger := zaptest.NewLogger(b)

	// Setup: create temp cert files
	tempDir := b.TempDir()
	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		b.Fatalf("Failed to generate cert: %v", err)
	}

	os.Setenv("TLS_ENABLED", "true")
	os.Setenv("TLS_CERT_FILE", certPath)
	os.Setenv("TLS_KEY_FILE", keyPath)
	defer func() {
		os.Unsetenv("TLS_ENABLED")
		os.Unsetenv("TLS_CERT_FILE")
		os.Unsetenv("TLS_KEY_FILE")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadTLSConfig(logger)
	}
}

// BenchmarkLoadTLSConfig_Disabled measures overhead when TLS is disabled
func BenchmarkLoadTLSConfig_Disabled(b *testing.B) {
	logger := zaptest.NewLogger(b)

	os.Setenv("TLS_ENABLED", "false")
	defer os.Unsetenv("TLS_ENABLED")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadTLSConfig(logger)
	}
}

// BenchmarkGenerateSelfSignedCert measures cert generation overhead
func BenchmarkGenerateSelfSignedCert(b *testing.B) {
	tempDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create unique subdirectory for each iteration
		iterDir := tempDir + "/iter" + string(rune(i))
		os.MkdirAll(iterDir, 0755)
		b.StartTimer()

		_, _, _ = GenerateSelfSignedCert(iterDir)
	}
}

// BenchmarkTLSHandshake measures TLS handshake overhead (simulation)
func BenchmarkTLSHandshake(b *testing.B) {
	tempDir := b.TempDir()
	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		b.Fatalf("Failed to generate cert: %v", err)
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		b.Fatalf("Failed to load cert: %v", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Clone()
	}
}

// BenchmarkLoadX509KeyPair measures certificate loading overhead
func BenchmarkLoadX509KeyPair(b *testing.B) {
	tempDir := b.TempDir()
	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		b.Fatalf("Failed to generate cert: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tls.LoadX509KeyPair(certPath, keyPath)
	}
}

// BenchmarkTLSConfig_Clone measures config clone overhead
func BenchmarkTLSConfig_Clone(b *testing.B) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Clone()
	}
}
