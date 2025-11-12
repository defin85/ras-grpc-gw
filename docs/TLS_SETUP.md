# TLS Setup Guide

This guide explains how to configure TLS for ras-grpc-gw to encrypt passwords and sensitive data in transit.

## Quick Start (Development)

For development/testing, ras-grpc-gw can automatically generate self-signed certificates:

```bash
export TLS_ENABLED=true
./ras-grpc-gw

# Server will auto-generate certs/dev-cert.pem and certs/dev-key.pem
```

**WARNING:** Self-signed certificates are for **DEVELOPMENT ONLY**. Do not use in production.

## Production Setup

### Option 1: Let's Encrypt (Recommended)

1. Install certbot:
```bash
# Ubuntu/Debian
sudo apt-get install certbot

# CentOS/RHEL
sudo yum install certbot
```

2. Generate certificate:
```bash
sudo certbot certonly --standalone -d your-domain.com
```

3. Configure ras-grpc-gw:
```bash
export TLS_ENABLED=true
export TLS_CERT_FILE=/etc/letsencrypt/live/your-domain.com/fullchain.pem
export TLS_KEY_FILE=/etc/letsencrypt/live/your-domain.com/privkey.pem

./ras-grpc-gw
```

4. Setup auto-renewal:
```bash
sudo certbot renew --dry-run
```

### Option 2: Custom Certificate Authority

1. Generate CA key and certificate:
```bash
# Generate CA private key
openssl genrsa -out ca-key.pem 4096

# Generate CA certificate
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
  -subj "/CN=My Company CA/O=My Company/C=US"
```

2. Generate server certificate:
```bash
# Generate server private key
openssl genrsa -out server-key.pem 2048

# Create certificate signing request
openssl req -new -key server-key.pem -out server.csr \
  -subj "/CN=ras-grpc-gw.example.com/O=My Company/C=US"

# Sign with CA
openssl x509 -req -days 365 -in server.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial \
  -out server-cert.pem
```

3. Configure ras-grpc-gw:
```bash
export TLS_ENABLED=true
export TLS_CERT_FILE=./server-cert.pem
export TLS_KEY_FILE=./server-key.pem

./ras-grpc-gw
```

### Option 3: Kubernetes Secrets

1. Create Kubernetes secret:
```bash
kubectl create secret tls ras-grpc-gw-tls \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem
```

2. Mount in deployment:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ras-grpc-gw
spec:
  template:
    spec:
      containers:
      - name: ras-grpc-gw
        image: ras-grpc-gw:latest
        env:
        - name: TLS_ENABLED
          value: "true"
        - name: TLS_CERT_FILE
          value: /etc/tls/tls.crt
        - name: TLS_KEY_FILE
          value: /etc/tls/tls.key
        volumeMounts:
        - name: tls
          mountPath: /etc/tls
          readOnly: true
      volumes:
      - name: tls
        secret:
          secretName: ras-grpc-gw-tls
```

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TLS_ENABLED` | No | `false` | Enable TLS encryption |
| `TLS_CERT_FILE` | Yes* | - | Path to certificate file (PEM format) |
| `TLS_KEY_FILE` | Yes* | - | Path to private key file (PEM format) |

\* Required when `TLS_ENABLED=true`. If not provided, self-signed cert will be auto-generated for development.

## Verifying TLS Configuration

### Test with grpcurl

```bash
# Without TLS (insecure)
grpcurl -plaintext localhost:50051 list

# With TLS and self-signed cert (dev)
grpcurl -insecure localhost:50051 list

# With TLS and trusted cert (production)
grpcurl -cacert ca-cert.pem ras-grpc-gw.example.com:50051 list
```

### Check Certificate Details

```bash
openssl x509 -in /path/to/cert.pem -text -noout
```

### Monitor TLS in Logs

```bash
# Server startup logs will show TLS status
INFO  TLS enabled for gRPC server
# or
WARN  TLS disabled - passwords transmitted in plaintext!
```

## Troubleshooting

### Error: "failed to load TLS certificate"

**Cause:** Certificate or key file not found, or invalid format.

**Solution:**
1. Verify files exist: `ls -la $TLS_CERT_FILE $TLS_KEY_FILE`
2. Check file permissions: `chmod 600 $TLS_KEY_FILE`
3. Validate certificate: `openssl x509 -in $TLS_CERT_FILE -text -noout`
4. Validate key: `openssl rsa -in $TLS_KEY_FILE -check`

### Error: "certificate signed by unknown authority"

**Cause:** Client doesn't trust the certificate authority.

**Solution (Development):**
```bash
# Use -insecure flag with grpcurl
grpcurl -insecure localhost:50051 list
```

**Solution (Production):**
1. Use Let's Encrypt (automatically trusted)
2. Or distribute your CA certificate to clients

### Error: "certificate has expired"

**Cause:** Certificate validity period has ended.

**Solution:**
1. Renew certificate (Let's Encrypt: `certbot renew`)
2. Update TLS_CERT_FILE path to new certificate
3. Restart ras-grpc-gw

## Security Best Practices

1. **Never use self-signed certificates in production**
   - Auto-generated certificates are for development only
   - Use Let's Encrypt or proper CA-signed certificates

2. **Protect private keys**
   ```bash
   chmod 600 /path/to/key.pem
   chown ras-grpc-gw:ras-grpc-gw /path/to/key.pem
   ```

3. **Rotate certificates regularly**
   - Set up automatic renewal (Let's Encrypt does this automatically)
   - Monitor certificate expiration dates

4. **Use strong cipher suites**
   - ras-grpc-gw enforces TLS 1.2+ with strong ciphers
   - No action needed - configured automatically

5. **Monitor TLS connections**
   - Check audit logs for TLS status
   - Alert on "TLS disabled" warnings in production

## Performance Impact

- **CPU overhead:** ~5-10% for TLS encryption/decryption
- **Latency:** +1-2ms per request
- **Memory:** +10-20MB for TLS buffers

**Recommendation:** TLS overhead is negligible compared to security benefits. Always enable TLS in production.

## Next Steps

- Configure [Audit Logging](AUDIT_LOGGING.md) to monitor operations
- Review [Security Best Practices](SECURITY.md)
- Set up certificate monitoring and auto-renewal
