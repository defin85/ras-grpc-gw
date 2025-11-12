// Package tlsconfig provides TLS configuration utilities for gRPC servers.
//
// # Overview
//
// This package simplifies TLS configuration for production gRPC deployments:
//   - Environment variable-based configuration
//   - Automatic generation of self-signed certificates for development
//   - Production-ready TLS settings (TLS 1.2+, strong cipher suites)
//
// # Quick Start
//
// For development with auto-generated self-signed certificates:
//
//	export TLS_ENABLED=true
//	./ras-grpc-gw
//
// The server will automatically generate a self-signed certificate in ./certs/
// and log a warning: "Using self-signed certificate (DEV ONLY)"
//
// For production with Let's Encrypt:
//
//	export TLS_ENABLED=true
//	export TLS_CERT_FILE=/etc/letsencrypt/live/domain.com/fullchain.pem
//	export TLS_KEY_FILE=/etc/letsencrypt/live/domain.com/privkey.pem
//	./ras-grpc-gw
//
// # Environment Variables
//
//	TLS_ENABLED       Enable or disable TLS (true/false)
//	TLS_CERT_FILE     Path to TLS certificate file (PEM format)
//	TLS_KEY_FILE      Path to TLS private key file (PEM format)
//
// # TLS Configuration
//
// The package enforces production-grade TLS settings:
//   - Minimum version: TLS 1.2
//   - Cipher suites: Only ECDHE (Perfect Forward Secrecy) with AEAD modes
//   - Supported ciphers:
//     * TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
//     * TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
//     * TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305
//
// # Self-Signed Certificates
//
// For development and testing, the package can automatically generate self-signed
// certificates with the following properties:
//   - Algorithm: RSA 2048-bit
//   - Validity: 365 days
//   - Common Name: localhost
//   - DNS Names: localhost
//   - IP Addresses: 127.0.0.1, ::1
//
// WARNING: Self-signed certificates should NEVER be used in production!
//
// Example usage:
//
//	import "github.com/khorevaa/ras-grpc-gw/pkg/tlsconfig"
//
//	logger := zap.NewProduction()
//	tlsConfig, err := tlsconfig.LoadTLSConfig(logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	var opts []grpc.ServerOption
//	if tlsConfig != nil {
//	    creds := credentials.NewTLS(tlsConfig)
//	    opts = append(opts, grpc.Creds(creds))
//	}
//
//	server := grpc.NewServer(opts...)
//
// # Production Deployment
//
// For production deployment, use one of the following:
//
// 1. Let's Encrypt (recommended for public-facing servers):
//
//	sudo certbot certonly --standalone -d your-domain.com
//	export TLS_CERT_FILE=/etc/letsencrypt/live/your-domain.com/fullchain.pem
//	export TLS_KEY_FILE=/etc/letsencrypt/live/your-domain.com/privkey.pem
//
// 2. Custom CA (for internal infrastructure):
//
//	export TLS_CERT_FILE=/path/to/server-cert.pem
//	export TLS_KEY_FILE=/path/to/server-key.pem
//
// 3. Kubernetes secrets:
//
//	apiVersion: v1
//	kind: Secret
//	metadata:
//	  name: tls-secret
//	type: kubernetes.io/tls
//	data:
//	  tls.crt: <base64-encoded-cert>
//	  tls.key: <base64-encoded-key>
//
// See docs/TLS_SETUP.md for detailed production setup instructions.
//
// # Security Considerations
//
//   - Always use TLS in production to protect passwords in transit
//   - Rotate certificates before expiration (recommended: 30 days before)
//   - Store private keys securely (chmod 600, never commit to git)
//   - Use Let's Encrypt for automatic certificate renewal
//   - Monitor certificate expiration with alerts
//
// # Performance
//
//   - Certificate loading: <10ms (done once at startup)
//   - TLS handshake overhead: ~10-20% (acceptable for security)
//   - No impact on steady-state performance after handshake
package tlsconfig
