# Contributing to ras-grpc-gw Fork

Руководство для разработчиков, желающих внести вклад в fork `defin85/ras-grpc-gw`.

**Repository:** https://github.com/defin85/ras-grpc-gw
**Document Version:** 1.0
**Last Updated:** 2025-01-17

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Setup](#development-setup)
3. [Code Style Guide](#code-style-guide)
4. [Testing Requirements](#testing-requirements)
5. [Commit Convention](#commit-convention)
6. [Pull Request Process](#pull-request-process)
7. [Code Review Guidelines](#code-review-guidelines)
8. [CI/CD Pipeline](#cicd-pipeline)
9. [Release Process](#release-process)

---

## Getting Started

### Prerequisites

**Required:**
- Go 1.24+
- Git 2.30+
- Make
- Docker 20.10+
- 1C RAC CLI (для integration tests)

**Recommended:**
- golangci-lint 1.55+
- GoLand или VSCode с Go extension
- Protocol Buffers compiler (protoc)

### Fork & Clone

```bash
# 1. Fork репозиторий на GitHub
# Перейти на https://github.com/defin85/ras-grpc-gw
# Нажать "Fork"

# 2. Клонировать ваш fork
git clone https://github.com/YOUR-USERNAME/ras-grpc-gw.git
cd ras-grpc-gw

# 3. Добавить upstream remote
git remote add upstream https://github.com/defin85/ras-grpc-gw.git

# 4. Verify remotes
git remote -v
# origin    https://github.com/YOUR-USERNAME/ras-grpc-gw.git (fetch)
# upstream  https://github.com/defin85/ras-grpc-gw.git (fetch)
```

### First Time Setup

```bash
# 1. Install dependencies
make deps

# 2. Install development tools
make install-tools

# 3. Generate protobuf code (if needed)
make proto-gen

# 4. Run tests to verify setup
make test

# 5. Run linter
make lint
```

**Expected output:**
```
✓ Dependencies installed
✓ Tools installed (golangci-lint, mockery, etc.)
✓ Protobuf code generated
✓ All tests passed (coverage > 70%)
✓ Linter passed (0 issues)
```

---

## Development Setup

### Project Structure

```
ras-grpc-gw/
├── cmd/                    # Entry points
│   └── main.go            # Application bootstrap
├── internal/              # Private application code
│   ├── server/           # gRPC server implementation
│   ├── adapter/          # RAC CLI adapter
│   ├── config/           # Configuration management
│   ├── health/           # Health check handlers
│   └── metrics/          # Prometheus metrics
├── pkg/                   # Public libraries (если нужны)
├── protos/                # Protobuf definitions
│   └── ras/              # RAS service API
├── tests/                 # Tests
│   ├── unit/             # Unit tests (alongside code)
│   ├── integration/      # Integration tests
│   └── e2e/              # End-to-end tests
├── config/                # Configuration files
│   └── config.yaml       # Default config
├── scripts/               # Utility scripts
├── deployments/           # Deployment configs
│   ├── docker/           # Dockerfiles
│   └── k8s/              # Kubernetes manifests
└── docs/                  # Documentation
```

### Development Workflow

```bash
# 1. Sync with upstream
git fetch upstream
git checkout main
git merge upstream/main

# 2. Create feature branch
git checkout -b feature/my-awesome-feature

# 3. Make changes
# ... edit code ...

# 4. Run tests locally
make test

# 5. Run linter
make lint

# 6. Fix issues if any
make fmt  # Auto-format code

# 7. Commit changes (см. Commit Convention)
git add .
git commit -m "feat(adapter): add connection pooling"

# 8. Push to your fork
git push origin feature/my-awesome-feature

# 9. Create Pull Request на GitHub
```

### Makefile Commands

```bash
# Build
make build              # Build binary
make docker-build       # Build Docker image

# Testing
make test               # Run unit tests
make test-integration   # Run integration tests
make test-e2e          # Run E2E tests
make coverage          # Generate coverage report
make coverage-html     # Open coverage report in browser

# Code Quality
make lint              # Run golangci-lint
make fmt               # Format code (gofmt, goimports)
make vet               # Run go vet
make staticcheck       # Run staticcheck

# Development
make run               # Run locally
make watch             # Run with hot reload (air)
make clean             # Clean build artifacts

# Protobuf
make proto-gen         # Generate Go code from .proto files
make proto-lint        # Lint protobuf files

# Tools
make install-tools     # Install dev tools
make deps              # Download dependencies
make tidy              # Run go mod tidy
```

---

## Code Style Guide

### Go Code Style

Мы следуем официальным [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**Основные правила:**

1. **Форматирование:** Используйте `gofmt` и `goimports`
   ```bash
   make fmt
   ```

2. **Naming Conventions:**
   ```go
   // Good
   type UserService struct {}
   func (s *UserService) GetUserByID(id int) (*User, error) {}

   // Bad
   type userService struct {}
   func (s *userService) get_user_by_id(id int) (*User, error) {}
   ```

3. **Error Handling:**
   ```go
   // Good - wrap errors with context
   if err != nil {
       return fmt.Errorf("failed to execute RAC command: %w", err)
   }

   // Bad - lose context
   if err != nil {
       return err
   }
   ```

4. **Context Usage:**
   ```go
   // Good - pass context as first parameter
   func (a *Adapter) ExecuteCommand(ctx context.Context, cmd string) error {
       // Use ctx for timeout, cancellation
   }

   // Bad - no context
   func (a *Adapter) ExecuteCommand(cmd string) error {}
   ```

5. **Logging:**
   ```go
   // Good - structured logging with zap
   logger.Info("Command executed",
       zap.String("command", cmd),
       zap.Duration("duration", duration),
       zap.Error(err),
   )

   // Bad - unstructured logging
   log.Printf("Command %s executed in %v, error: %v", cmd, duration, err)
   ```

### Protobuf Style

```protobuf
// Good
syntax = "proto3";

package ras.v1;

option go_package = "github.com/defin85/ras-grpc-gw/pkg/api/ras/v1";

import "google/protobuf/timestamp.proto";

// ClusterInfo represents information about 1C cluster
message ClusterInfo {
  // Unique cluster identifier
  string id = 1;
  // Cluster name
  string name = 2;
  // Creation timestamp
  google.protobuf.Timestamp created_at = 3;
}
```

### Comment Style

```go
// Good - godoc style comments

// ExecuteCommand executes a RAC CLI command with timeout and retry logic.
//
// It wraps the rac CLI binary and handles:
//   - Command timeout (default 30s)
//   - Exponential backoff retry (max 3 attempts)
//   - Stderr parsing for errors
//
// Returns the command output or an error if execution fails.
func (a *Adapter) ExecuteCommand(ctx context.Context, cmd string) (string, error) {
    // Implementation
}

// Bad - redundant or missing comments

// ExecuteCommand executes command
func (a *Adapter) ExecuteCommand(ctx context.Context, cmd string) (string, error) {}
```

---

## Testing Requirements

### Test Coverage

**Минимальные требования:**
- Overall coverage: **> 70%**
- New code coverage: **> 80%**
- Critical paths coverage: **100%**

**Проверка:**
```bash
make coverage
# coverage: 72.5% of statements
```

### Unit Tests

**Расположение:** Рядом с кодом (`*_test.go`)

**Пример:**
```go
// internal/adapter/rac_test.go
package adapter

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestAdapter_ExecuteCommand_Success(t *testing.T) {
    // Arrange
    mockExec := new(MockExecutor)
    mockExec.On("Exec", mock.Anything, "rac", []string{"cluster", "list"}).
        Return("cluster1\ncluster2", nil)

    adapter := NewAdapter("/usr/bin/rac")
    adapter.executor = mockExec

    // Act
    output, err := adapter.ExecuteCommand(context.Background(), "cluster list")

    // Assert
    assert.NoError(t, err)
    assert.Contains(t, output, "cluster1")
    mockExec.AssertExpectations(t)
}

func TestAdapter_ExecuteCommand_Timeout(t *testing.T) {
    // Arrange
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()

    adapter := NewAdapter("/usr/bin/rac")

    // Act
    _, err := adapter.ExecuteCommand(ctx, "cluster list")

    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "context deadline exceeded")
}
```

**Запуск:**
```bash
# Все unit tests
go test ./...

# Конкретный package
go test ./internal/adapter -v

# С coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

**Расположение:** `tests/integration/`

**Пример:**
```go
// tests/integration/rac_integration_test.go
// +build integration

package integration

import (
    "context"
    "testing"
    "github.com/stretchr/testify/require"
)

func TestRAC_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Requires real RAC server (Docker Compose)
    adapter := adapter.NewAdapter("/usr/bin/rac")

    ctx := context.Background()
    output, err := adapter.ExecuteCommand(ctx, "cluster list --server=localhost:1545")

    require.NoError(t, err)
    require.NotEmpty(t, output)
}
```

**Запуск:**
```bash
# С Docker Compose (mock RAC server)
make test-integration

# Вручную
go test -tags=integration ./tests/integration/... -v
```

### E2E Tests

**Расположение:** `tests/e2e/`

**Запуск:**
```bash
make test-e2e
```

---

## Commit Convention

Мы используем **Conventional Commits 1.0.0**: https://www.conventionalcommits.org/

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | Новая feature | `feat(adapter): add connection pooling` |
| `fix` | Исправление бага | `fix(server): prevent goroutine leak on shutdown` |
| `docs` | Изменения в документации | `docs: update CONTRIBUTING.md` |
| `style` | Форматирование кода | `style: run gofmt on all files` |
| `refactor` | Рефакторинг | `refactor(config): simplify viper initialization` |
| `test` | Добавление тестов | `test(adapter): add unit tests for retry logic` |
| `chore` | Build, tooling | `chore: update golangci-lint to v1.55` |
| `perf` | Оптимизация производительности | `perf(adapter): reduce allocations in exec` |

### Scopes

- `adapter` - RAC adapter
- `server` - gRPC server
- `config` - Configuration
- `metrics` - Prometheus metrics
- `health` - Health checks
- `docker` - Docker images
- `k8s` - Kubernetes manifests
- `ci` - CI/CD pipeline

### Examples

**Good commits:**
```
feat(adapter): implement connection pooling for RAC CLI

Adds a connection pool to limit concurrent RAC CLI executions
to prevent server overload. Configurable via max_connections.

Closes #42

---

fix(server): graceful shutdown not waiting for requests

Previously, shutdown would immediately kill in-flight requests.
Now waits up to 30 seconds (configurable) for completion.

Fixes #57

---

docs: add production deployment guide

Comprehensive guide covering:
- Docker deployment
- Kubernetes deployment
- Monitoring setup
- Troubleshooting

---

test(adapter): increase coverage to 85%

Added unit tests for:
- Timeout handling
- Retry logic with exponential backoff
- Error parsing from stderr
```

**Bad commits:**
```
update code          # Слишком общее
fixed bug            # Нет контекста
WIP                  # Не commit в main/PR
oops                 # Не информативно
```

---

## Pull Request Process

### Before Creating PR

**Checklist:**
- [ ] Код прошёл `make lint` без ошибок
- [ ] Все тесты проходят (`make test`)
- [ ] Coverage > 70% (проверить `make coverage`)
- [ ] Документация обновлена (если нужно)
- [ ] CHANGELOG.md обновлён (для features/fixes)
- [ ] Commits следуют Conventional Commits

### Creating PR

```bash
# 1. Push feature branch
git push origin feature/my-awesome-feature

# 2. Create PR через GitHub UI или CLI
gh pr create \
  --title "feat(adapter): add connection pooling" \
  --body "$(cat <<'EOF'
## Summary

Implements connection pooling for RAC CLI to prevent server overload.

## Changes

- Add `ConnectionPool` struct with semaphore-based pooling
- Configurable max connections (default: 10)
- Update config schema to include `max_connections`
- Add unit tests (coverage +15%)

## Testing

- [x] Unit tests pass
- [x] Integration tests pass
- [x] Manually tested with 50 concurrent requests
- [x] golangci-lint clean

## Breaking Changes

None

## Related Issues

Closes #42

## Checklist

- [x] Lint passed
- [x] Tests pass (coverage > 70%)
- [x] Documentation updated
- [x] CHANGELOG.md updated
EOF
)" \
  --base main \
  --head feature/my-awesome-feature
```

### PR Template

```markdown
## Summary

<!-- Краткое описание изменений -->

## Changes

<!-- Детальный список изменений -->
- Change 1
- Change 2

## Testing

<!-- Как тестировали -->
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing performed
- [ ] golangci-lint clean

## Breaking Changes

<!-- Есть ли breaking changes? Как мигрировать? -->

## Related Issues

<!-- Closes #123, Fixes #456 -->

## Screenshots (if applicable)

<!-- Для UI changes или grafana dashboards -->

## Checklist

- [ ] Lint passed (`make lint`)
- [ ] Tests pass (`make test`)
- [ ] Coverage > 70% (`make coverage`)
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Commits follow Conventional Commits
```

### PR Size Guidelines

**Preferred:**
- Small PRs: < 300 lines changed
- Single responsibility
- Easy to review

**Acceptable:**
- Medium PRs: 300-800 lines
- Clear scope
- Well-documented

**Avoid:**
- Large PRs: > 800 lines
- Multiple unrelated changes
- Hard to review

**Tip:** Разбивайте большие features на несколько PRs:
```
PR #1: feat(adapter): add ConnectionPool struct (foundation)
PR #2: feat(adapter): integrate ConnectionPool in Adapter
PR #3: feat(config): add max_connections configuration
PR #4: test(adapter): add integration tests for pooling
```

---

## Code Review Guidelines

### For Authors

**Responding to feedback:**
- ✅ Будьте открыты к конструктивной критике
- ✅ Объясняйте свои решения если reviewer не понял
- ✅ Применяйте suggestions если согласны
- ✅ Отмечайте resolved conversations

**Updating PR:**
```bash
# Make requested changes
# ... edit code ...

# Commit changes
git add .
git commit -m "refactor(adapter): apply review suggestions"

# Push to same branch
git push origin feature/my-awesome-feature
# PR автоматически обновится
```

### For Reviewers

**Review focus areas:**
1. **Correctness:** Код делает то, что заявлено?
2. **Testing:** Достаточно тестов? Coverage > 70%?
3. **Code Quality:** Читаемый? Поддерживаемый? Следует style guide?
4. **Performance:** Нет очевидных bottlenecks?
5. **Security:** Нет уязвимостей? (SQL injection, command injection)
6. **Documentation:** Достаточно комментариев? Обновлена документация?

**Review etiquette:**
- 🟢 **Good:** "Consider using `context.WithTimeout()` here to prevent hanging"
- 🔴 **Bad:** "This is wrong"

**Comment prefixes:**
- `nit:` - Minor suggestion, не блокирующее
- `question:` - Вопрос для понимания
- `suggestion:` - Предложение улучшения
- `blocker:` - Критическая проблема, должна быть исправлена

**Example comments:**
```
nit: Consider extracting this logic into a separate function for reusability.

---

question: Why did you choose exponential backoff over linear?
Any performance benchmarks?

---

suggestion: You could simplify this with `errors.Is()` instead of string comparison.

if errors.Is(err, ErrNotFound) {
    // ...
}

---

blocker: This will cause a goroutine leak because the channel is never closed.
Need to add `close(ch)` after the loop.
```

### Approval Process

**Requirements для merge:**
- ✅ Минимум 1 approval от maintainer
- ✅ Все CI checks зелёные
- ✅ No unresolved conversations
- ✅ Branch up-to-date с main

**Merge strategies:**
- **Squash and merge** (preferred): Для feature branches
- **Rebase and merge**: Для hotfixes
- **Merge commit**: НЕ используется

---

## CI/CD Pipeline

### GitHub Actions Workflows

#### 1. Continuous Integration (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main`, `develop`
- Pull Request to `main`

**Jobs:**
```yaml
jobs:
  lint:
    - golangci-lint run
    - buf lint (protobuf)

  test:
    - go test -race -coverprofile=coverage.out ./...
    - coverage gate (min 70%)

  build:
    - go build ./cmd/...
    - docker build

  integration-test:
    - docker-compose up (mock RAC server)
    - go test -tags=integration ./tests/integration/...
```

**Expected duration:** ~5-10 minutes

#### 2. Release (`.github/workflows/release.yml`)

**Triggers:**
- Push tag `v*` (e.g., `v1.0.0-cc`)

**Jobs:**
```yaml
jobs:
  release:
    - GoReleaser (multi-platform binaries)
    - Docker image build + push to GHCR
    - GitHub Release creation
```

### Pre-commit Hooks (Recommended)

```bash
# Install pre-commit framework
pip install pre-commit

# Install hooks
cat > .pre-commit-config.yaml <<'EOF'
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: go fmt
        entry: make fmt
        language: system
        pass_filenames: false

      - id: go-lint
        name: golangci-lint
        entry: make lint
        language: system
        pass_filenames: false

      - id: go-test
        name: go test
        entry: make test
        language: system
        pass_filenames: false
EOF

pre-commit install

# Теперь hooks запустятся автоматически перед каждым commit
```

---

## Release Process

### Versioning

Semantic Versioning 2.0.0 с suffix `-cc`:

```
vMAJOR.MINOR.PATCH-cc

Examples:
- v1.0.0-cc - первый production релиз
- v1.1.0-cc - новые features (обратно совместимые)
- v1.0.1-cc - bugfix
- v2.0.0-cc - breaking changes
```

### Release Checklist

**Pre-release (за 1-2 дня до релиза):**
- [ ] Все запланированные features/fixes merged
- [ ] CHANGELOG.md обновлён (Unreleased → версия)
- [ ] Документация актуальна
- [ ] Security audit пройден (Dependabot, Trivy)
- [ ] Staging deployment протестирован

**Release day:**
```bash
# 1. Создать release branch
git checkout -b release/v1.0.0-cc main

# 2. Update CHANGELOG.md
# Переместить "Unreleased" в "v1.0.0-cc - 2025-01-17"

# 3. Bump version
echo "v1.0.0-cc" > VERSION

# 4. Commit
git commit -am "chore: release v1.0.0-cc"

# 5. Push + PR
git push origin release/v1.0.0-cc
gh pr create --title "Release v1.0.0-cc" --base main

# 6. Merge PR после review

# 7. Создать Git tag
git checkout main
git pull
git tag -a v1.0.0-cc -m "Release v1.0.0-cc"
git push origin v1.0.0-cc

# 8. GitHub Actions автоматически создаст release
```

**Post-release:**
- [ ] Verify release artifacts (binaries, Docker image)
- [ ] Test Docker image: `docker run ghcr.io/defin85/ras-grpc-gw:v1.0.0-cc --version`
- [ ] Announce release (Slack, email, etc.)
- [ ] Update production deployment

---

## Questions?

**Документация:**
- [FORK_AUDIT.md](./FORK_AUDIT.md) - Audit upstream
- [FORK_CHANGELOG.md](./FORK_CHANGELOG.md) - История изменений
- [UPSTREAM_SYNC.md](./UPSTREAM_SYNC.md) - Синхронизация с upstream
- [PRODUCTION_GUIDE.md](./PRODUCTION_GUIDE.md) - Production deployment

**Связь:**
- GitHub Issues: https://github.com/defin85/ras-grpc-gw/issues
- GitHub Discussions: https://github.com/defin85/ras-grpc-gw/discussions
- CommandCenter1C Team: TBD

---

**Document Version:** 1.0
**Last Updated:** 2025-01-17
**Next Review:** 2025-02-17
