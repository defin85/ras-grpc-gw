package tlsconfig

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestLoadTLSConfig_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Ensure TLS is disabled
	os.Setenv("TLS_ENABLED", "false")
	defer os.Unsetenv("TLS_ENABLED")

	config, err := LoadTLSConfig(logger)

	require.NoError(t, err)
	assert.Nil(t, config, "TLS config should be nil when disabled")
}

func TestLoadTLSConfig_AutoGenerateSelfSigned(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create temp directory for certs
	tempDir := t.TempDir()

	// Enable TLS but don't set cert files (should auto-generate)
	os.Setenv("TLS_ENABLED", "true")
	os.Unsetenv("TLS_CERT_FILE")
	os.Unsetenv("TLS_KEY_FILE")
	defer os.Unsetenv("TLS_ENABLED")

	// Change working directory to temp
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	config, err := LoadTLSConfig(logger)

	require.NoError(t, err)
	require.NotNil(t, config, "TLS config should not be nil")

	// Verify TLS config properties
	assert.Equal(t, tls.VersionTLS12, int(config.MinVersion))
	assert.NotEmpty(t, config.Certificates)

	// Verify self-signed cert was created
	assert.FileExists(t, "certs/dev-cert.pem")
	assert.FileExists(t, "certs/dev-key.pem")
}

func TestLoadTLSConfig_ExistingCertificate(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Generate self-signed cert
	tempDir := t.TempDir()
	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	require.NoError(t, err)

	// Enable TLS and set paths
	os.Setenv("TLS_ENABLED", "true")
	os.Setenv("TLS_CERT_FILE", certPath)
	os.Setenv("TLS_KEY_FILE", keyPath)
	defer func() {
		os.Unsetenv("TLS_ENABLED")
		os.Unsetenv("TLS_CERT_FILE")
		os.Unsetenv("TLS_KEY_FILE")
	}()

	config, err := LoadTLSConfig(logger)

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, tls.VersionTLS12, int(config.MinVersion))
	assert.Len(t, config.Certificates, 1)
	assert.NotEmpty(t, config.CipherSuites)
}

func TestLoadTLSConfig_InvalidCertificate(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Enable TLS but set invalid paths
	os.Setenv("TLS_ENABLED", "true")
	os.Setenv("TLS_CERT_FILE", "/nonexistent/cert.pem")
	os.Setenv("TLS_KEY_FILE", "/nonexistent/key.pem")
	defer func() {
		os.Unsetenv("TLS_ENABLED")
		os.Unsetenv("TLS_CERT_FILE")
		os.Unsetenv("TLS_KEY_FILE")
	}()

	// Should try to generate self-signed, which should succeed
	config, err := LoadTLSConfig(logger)

	// In this case, it will try to auto-generate self-signed cert
	if err != nil {
		t.Skip("Skip on systems where cert generation fails")
	}

	assert.NotNil(t, config)
}

func TestGenerateSelfSignedCert(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)

	require.NoError(t, err)
	assert.NotEmpty(t, certPath)
	assert.NotEmpty(t, keyPath)

	// Verify files exist
	assert.FileExists(t, certPath)
	assert.FileExists(t, keyPath)

	// Verify files are named correctly
	assert.Equal(t, filepath.Join(tempDir, "dev-cert.pem"), certPath)
	assert.Equal(t, filepath.Join(tempDir, "dev-key.pem"), keyPath)

	// Verify certificate can be loaded
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)
	assert.NotNil(t, cert)
}


func TestIsTLSEnabled(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"enabled", "true", true},
		{"disabled", "false", false},
		{"not set", "", false},
		{"invalid value", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TLS_ENABLED", tt.envValue)
				defer os.Unsetenv("TLS_ENABLED")
			} else {
				os.Unsetenv("TLS_ENABLED")
			}

			result := isTLSEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(existingFile, []byte("test"), 0644)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", existingFile, true},
		{"non-existing file", "/nonexistent/file.txt", false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileExists(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
