# Audit Logging Guide

This guide explains the audit logging system in ras-grpc-gw and how to integrate it with log aggregation platforms.

## Overview

ras-grpc-gw logs all gRPC operations in structured JSON format for security monitoring, compliance, and debugging.

## Log Format

All audit logs use structured JSON with the following fields:

```json
{
  "timestamp": "2025-11-02T15:30:45.123Z",
  "level": "info",
  "operation": "/infobase.service.InfobaseManagementService/CreateInfobase",
  "cluster_id": "uuid-cluster-123",
  "infobase_id": "uuid-infobase-456",
  "user": "admin",
  "result": "success",
  "duration_ms": 1234
}
```

## Log Levels

- **INFO:** Successful operations
- **WARN:** Destructive operations (DropInfobase)
- **ERROR:** Failed operations

## Example Logs

### Successful Operation
```json
{
  "level": "info",
  "operation": "/infobase.service.InfobaseManagementService/CreateInfobase",
  "cluster_id": "uuid-123",
  "user": "admin",
  "result": "success",
  "duration_ms": 1234
}
```

### Destructive Operation (Warning)
```json
{
  "level": "warn",
  "operation": "/infobase.service.InfobaseManagementService/DropInfobase",
  "cluster_id": "uuid-123",
  "infobase_id": "uuid-456",
  "user": "admin",
  "result": "success",
  "duration_ms": 2345
}
```

### Failed Operation
```json
{
  "level": "error",
  "operation": "/infobase.service.InfobaseManagementService/UpdateInfobase",
  "error": "infobase not found",
  "grpc_code": "NotFound",
  "duration_ms": 123
}
```

## Integration with ELK Stack

### 1. Filebeat Configuration

```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/ras-grpc-gw/*.log
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "ras-grpc-gw-%{+yyyy.MM.dd}"
```

### 2. Logstash Configuration

```ruby
input {
  file {
    path => "/var/log/ras-grpc-gw/*.log"
    codec => json
  }
}

filter {
  if [operation] {
    mutate {
      add_field => { "service" => "ras-grpc-gw" }
    }
  }
}

output {
  elasticsearch {
    hosts => ["localhost:9200"]
    index => "ras-grpc-gw-%{+YYYY.MM.dd}"
  }
}
```

### 3. Kibana Dashboard

Create visualizations for:
- Operations per minute (line chart)
- Error rate (gauge)
- Slow operations (table: duration_ms > 5000)
- Destructive operations (filtered by level:warn)
- User activity (pie chart by user)

## Integration with Grafana Loki

### Promtail Configuration

```yaml
server:
  http_listen_port: 9080

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: ras-grpc-gw
    static_configs:
    - targets:
        - localhost
      labels:
        job: ras-grpc-gw
        __path__: /var/log/ras-grpc-gw/*.log
    pipeline_stages:
    - json:
        expressions:
          operation: operation
          user: user
          result: result
          duration_ms: duration_ms
```

### Grafana Queries

```logql
# All errors
{job="ras-grpc-gw"} | json | level="error"

# Destructive operations
{job="ras-grpc-gw"} | json | level="warn"

# Slow operations (>5s)
{job="ras-grpc-gw"} | json | duration_ms > 5000

# Operations by user
sum by (user) (count_over_time({job="ras-grpc-gw"}[1h]))
```

## Monitoring Alerts

### High Error Rate
```yaml
alert: HighErrorRate
expr: rate(ras_grpc_gw_errors_total[5m]) > 0.1
annotations:
  summary: "Error rate > 10%"
```

### Slow Operations
```yaml
alert: SlowOperations
expr: ras_grpc_gw_duration_ms > 10000
annotations:
  summary: "Operation took > 10s"
```

### Destructive Operations
```yaml
alert: DestructiveOperation
expr: ras_grpc_gw_destructive_ops_total > 0
annotations:
  summary: "DropInfobase operation executed"
```

## Log Retention

Recommended retention policies:
- **Hot storage:** 7 days (for active investigations)
- **Warm storage:** 90 days (for compliance)
- **Cold storage:** 1 year (for audit requirements)

## Security Considerations

1. **Protect log files:** `chmod 600 /var/log/ras-grpc-gw/*.log`
2. **Encrypt logs at rest:** Use encrypted filesystems
3. **Restrict access:** Only security team should access audit logs
4. **Monitor log integrity:** Detect tampering attempts

## Additional Resources

- [Security Guide](SECURITY.md)
- [TLS Setup](TLS_SETUP.md)
