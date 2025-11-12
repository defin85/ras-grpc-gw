package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestGenerateSelfSignedCert_CertificateValidity tests certificate validity period
func TestGenerateSelfSignedCert_CertificateValidity(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Load and parse certificate
	cert, err := loadCertificate(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	// Check validity period (should be 365 days)
	notBefore := cert.NotBefore
	notAfter := cert.NotAfter
	validityDays := notAfter.Sub(notBefore).Hours() / 24

	if validityDays < 364 || validityDays > 366 {
		t.Errorf("Certificate validity period = %.0f days, want 365 days", validityDays)
	}

	t.Logf("Certificate valid from %v to %v (%.0f days)", notBefore, notAfter, validityDays)
}

// TestGenerateSelfSignedCert_DNSNames tests certificate DNS names
func TestGenerateSelfSignedCert_DNSNames(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Load and parse certificate
	cert, err := loadCertificate(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	// Check DNS names
	expectedDNSNames := []string{"localhost"}
	if len(cert.DNSNames) != len(expectedDNSNames) {
		t.Errorf("Certificate has %d DNS names, want %d", len(cert.DNSNames), len(expectedDNSNames))
	}

	for _, expected := range expectedDNSNames {
		found := false
		for _, dnsName := range cert.DNSNames {
			if dnsName == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Certificate missing DNS name: %s", expected)
		}
	}

	t.Logf("Certificate DNS names: %v", cert.DNSNames)
}

// TestGenerateSelfSignedCert_IPAddresses tests certificate IP addresses
func TestGenerateSelfSignedCert_IPAddresses(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Load and parse certificate
	cert, err := loadCertificate(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	// Check IP addresses (should have 127.0.0.1 and ::1)
	if len(cert.IPAddresses) < 2 {
		t.Errorf("Certificate has %d IP addresses, want at least 2 (IPv4 + IPv6)", len(cert.IPAddresses))
	}

	hasIPv4 := false
	hasIPv6 := false

	for _, ip := range cert.IPAddresses {
		if ip.String() == "127.0.0.1" {
			hasIPv4 = true
		}
		if ip.String() == "::1" {
			hasIPv6 = true
		}
	}

	if !hasIPv4 {
		t.Error("Certificate missing IPv4 loopback address (127.0.0.1)")
	}
	if !hasIPv6 {
		t.Error("Certificate missing IPv6 loopback address (::1)")
	}

	t.Logf("Certificate IP addresses: %v", cert.IPAddresses)
}

// TestGenerateSelfSignedCert_KeyUsage tests certificate key usage
func TestGenerateSelfSignedCert_KeyUsage(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Load and parse certificate
	cert, err := loadCertificate(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	// Check key usage (should include digital signature and key encipherment)
	expectedUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment

	if cert.KeyUsage&expectedUsage != expectedUsage {
		t.Errorf("Certificate key usage = %v, want to include %v", cert.KeyUsage, expectedUsage)
	}

	t.Logf("Certificate key usage: %v", cert.KeyUsage)
}

// TestGenerateSelfSignedCert_FilePermissions tests that cert files have correct permissions
func TestGenerateSelfSignedCert_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Check cert file permissions (should be readable)
	certInfo, err := os.Stat(certPath)
	if err != nil {
		t.Fatalf("Failed to stat cert file: %v", err)
	}

	certMode := certInfo.Mode()
	if certMode&0444 == 0 {
		t.Errorf("Certificate file not readable: %v", certMode)
	}

	// Check key file permissions (should be readable - note: in real production, key should be restricted)
	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	keyMode := keyInfo.Mode()
	if keyMode&0400 == 0 {
		t.Errorf("Key file not readable: %v", keyMode)
	}

	t.Logf("Certificate file mode: %v", certMode)
	t.Logf("Key file mode: %v", keyMode)
}

// TestGenerateSelfSignedCert_FileContent tests that generated files contain valid PEM data
func TestGenerateSelfSignedCert_FileContent(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Check cert file content
	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("Failed to read cert file: %v", err)
	}

	if len(certData) == 0 {
		t.Error("Certificate file is empty")
	}

	// PEM cert should start with -----BEGIN CERTIFICATE-----
	certPEM := string(certData)
	if len(certPEM) < 27 || certPEM[:27] != "-----BEGIN CERTIFICATE-----" {
		t.Error("Certificate file does not contain valid PEM data")
	}

	// Check key file content
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	if len(keyData) == 0 {
		t.Error("Key file is empty")
	}

	// PEM key should start with -----BEGIN RSA PRIVATE KEY-----
	keyPEM := string(keyData)
	if len(keyPEM) < 31 || keyPEM[:31] != "-----BEGIN RSA PRIVATE KEY-----" {
		t.Error("Key file does not contain valid PEM data")
	}

	t.Logf("Certificate file size: %d bytes", len(certData))
	t.Logf("Key file size: %d bytes", len(keyData))
}

// TestGenerateSelfSignedCert_FileLocations tests that files are created in correct locations
func TestGenerateSelfSignedCert_FileLocations(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	// Check that returned paths are correct (using dev-cert.pem and dev-key.pem)
	expectedCertPath := filepath.Join(tempDir, "dev-cert.pem")
	expectedKeyPath := filepath.Join(tempDir, "dev-key.pem")

	if certPath != expectedCertPath {
		t.Errorf("certPath = %s, want %s", certPath, expectedCertPath)
	}

	if keyPath != expectedKeyPath {
		t.Errorf("keyPath = %s, want %s", keyPath, expectedKeyPath)
	}

	// Verify files actually exist at those locations
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Errorf("Certificate file does not exist at %s", certPath)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("Key file does not exist at %s", keyPath)
	}
}

// TestGenerateSelfSignedCert_MultipleCallsSameDir tests that multiple calls to same directory succeed
func TestGenerateSelfSignedCert_MultipleCallsSameDir(t *testing.T) {
	tempDir := t.TempDir()

	// First call
	certPath1, keyPath1, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("First GenerateSelfSignedCert failed: %v", err)
	}

	// Second call - should overwrite existing files
	certPath2, keyPath2, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("Second GenerateSelfSignedCert failed: %v", err)
	}

	// Paths should be the same
	if certPath1 != certPath2 {
		t.Errorf("certPath changed between calls: %s != %s", certPath1, certPath2)
	}

	if keyPath1 != keyPath2 {
		t.Errorf("keyPath changed between calls: %s != %s", keyPath1, keyPath2)
	}

	// Files should still be valid
	_, err = loadCertificate(certPath2, keyPath2)
	if err != nil {
		t.Fatalf("Failed to load certificate after second generation: %v", err)
	}
}

// TestGenerateSelfSignedCert_ReadOnlyDirectory tests error handling when directory becomes read-only (Unix only)
func TestGenerateSelfSignedCert_ReadOnlyDirectory(t *testing.T) {
	// Skip on Windows as chmod doesn't work the same way
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific permission test on Windows")
	}

	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()

	// Make directory read-only
	if err := os.Chmod(tempDir, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(tempDir, 0755) // Restore permissions for cleanup

	_, _, err := GenerateSelfSignedCert(tempDir)

	// Should fail on Unix systems
	if err == nil {
		t.Error("Expected error for read-only directory, got nil")
	} else {
		t.Logf("Got expected permission error: %v", err)
	}
}

// TestGenerateSelfSignedCert_CommonName tests certificate common name
func TestGenerateSelfSignedCert_CommonName(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	cert, err := loadCertificate(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	expectedCN := "localhost"
	if cert.Subject.CommonName != expectedCN {
		t.Errorf("Certificate CommonName = %s, want %s", cert.Subject.CommonName, expectedCN)
	}

	t.Logf("Certificate CommonName: %s", cert.Subject.CommonName)
}

// TestGenerateSelfSignedCert_Organization tests certificate organization
func TestGenerateSelfSignedCert_Organization(t *testing.T) {
	tempDir := t.TempDir()

	certPath, keyPath, err := GenerateSelfSignedCert(tempDir)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert failed: %v", err)
	}

	cert, err := loadCertificate(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	if len(cert.Subject.Organization) == 0 {
		t.Error("Certificate has no organization")
	}

	expectedOrg := "ras-grpc-gw Development"
	if cert.Subject.Organization[0] != expectedOrg {
		t.Errorf("Certificate Organization = %s, want %s", cert.Subject.Organization[0], expectedOrg)
	}

	t.Logf("Certificate Organization: %v", cert.Subject.Organization)
}

// Helper function to load certificate
func loadCertificate(certPath, keyPath string) (*x509.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}

	return x509Cert, nil
}
