# Security Features

This document describes the security mechanisms implemented in ras-grpc-gw.

## Overview

ras-grpc-gw implements three layers of security:

1. **Password Sanitization** - Automatic redaction of passwords in logs
2. **TLS Encryption** - Encrypted transport for passwords in transit
3. **Audit Logging** - Complete audit trail of all operations

## Password Sanitization

The password sanitization interceptor uses protobuf reflection to automatically detect and sanitize all password fields in gRPC requests before logging.

**Detection Pattern:** Any protobuf field ending with `*_password`

**Replacement:**
- Non-empty passwords → `"******"`
- Empty passwords → `""` (unchanged)

### Important Notes

- **Original request is NOT modified** - only the logged version is sanitized
- **Works automatically** - no manual intervention required
- **Covers all gRPC methods** - protobuf reflection ensures complete coverage

## TLS Encryption

See [TLS Setup Guide](TLS_SETUP.md) for detailed configuration.

**Minimum TLS Version:** 1.2

**Cipher Suites:**
- TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
- TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
- TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305

## Audit Logging

Every gRPC request is logged with timestamp, operation, user, cluster/infobase IDs, result, and duration.

See [Audit Logging Guide](AUDIT_LOGGING.md) for details.

## Security Best Practices

1. **Always Enable TLS in Production**
2. **Protect Private Keys** - `chmod 600 /path/to/key.pem`
3. **Rotate Credentials Regularly**
4. **Monitor Audit Logs** for suspicious activity
5. **Secure Log Storage** - encrypt logs at rest

## Compliance

- **SOC 2:** Audit logging (CC7.2), TLS encryption (CC6.1)
- **ISO 27001:** Access control (A.9.4.1), Cryptographic controls (A.10.1.1)
- **GDPR:** Data access tracking (Article 30), Data protection (Article 32)

## Additional Resources

- [TLS Setup Guide](TLS_SETUP.md)
- [Audit Logging Guide](AUDIT_LOGGING.md)
