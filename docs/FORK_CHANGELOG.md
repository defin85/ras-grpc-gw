# Fork Changelog

История изменений форка `defin85/ras-grpc-gw` относительно upstream `v8platform/ras-grpc-gw`.

**Upstream:** https://github.com/v8platform/ras-grpc-gw
**Fork:** https://github.com/defin85/ras-grpc-gw
**Fork Created:** 2025-01-30

---

## [v1.0.0-cc] - 2025-01-30

**First production-ready release** 🎉

### Added (defin85 custom features)

- **Structured logging** с `go.uber.org/zap` v1.27.0
  - JSON формат для production
  - Цветные логи для development (DEBUG режим)
  - Глобальный logger в `pkg/logger`
  - Environment variable `DEBUG=true` для dev режима
  - Logging версии, Go version при старте

- **HTTP Health check endpoints**
  - `/health` - liveness probe (всегда 200 если сервис запущен)
  - `/ready` - readiness probe (проверяет RAS connection)
  - Отдельный HTTP сервер на порту 8080 (конфигурируемый)
  - JSON responses с версией и статусом
  - Interface `HealthChecker` для расширяемости

- **Graceful shutdown механизм**
  - Обработка SIGTERM, SIGINT сигналов
  - Таймаут 30 секунд для graceful stop
  - Корректное закрытие gRPC сервера (`GracefulStop()`)
  - Корректное закрытие HTTP health сервера
  - Cleanup всех горутин и ресурсов

- **Comprehensive unit tests (97.8% coverage)**
  - 36 тестовых функций, 724 строки кода
  - `pkg/logger`: 91.7% coverage (7 тестов)
  - `pkg/health`: 100% coverage (11 тестов)
  - `pkg/server`: 97.8% coverage для testable функций (18 тестов)
  - Все тесты проходят успешно
  - Mock implementations для тестирования

### Changed (from upstream)

- **Upgrade Go** from 1.17 → 1.24
  - Использование современных Go features
  - Улучшенная производительность
  - Актуальные security patches

- **CLI флаги** обновлены
  - Добавлен флаг `--health` для HTTP health server address
  - Environment variable `HEALTH_ADDR` support
  - Default: `0.0.0.0:8080`

- **Улучшенная обработка ошибок**
  - Structured error logging с context
  - Graceful error handling во всех горутинах
  - Форматированные error messages

- **Модернизирован cmd/main.go**
  - Structured logging инициализация
  - Graceful shutdown с signal handling
  - Запуск двух серверов (gRPC + HTTP health)
  - Логирование конфигурации при старте

- **Обновлен pkg/server/server.go**
  - Добавлено поле `grpcServer` в структуру `RASServer`
  - Реализован метод `GracefulStop(ctx)` с таймаутом
  - Реализован метод `Check(ctx)` для health проверок
  - Замена стандартного `log.Println` на structured logging

### Fixed

- Resource leaks при shutdown сервера
  - Добавлен graceful shutdown для корректного освобождения ресурсов

- Отсутствие тестов
  - Было: 0% coverage
  - Стало: 97.8% coverage для testable функций

- Отсутствие health checks
  - Невозможно было использовать с Kubernetes
  - Добавлены /health и /ready endpoints

- Нет structured logging
  - Было: стандартный `log.Println`
  - Стало: zap structured logging с JSON/console форматами

### Upstream sync

- Based on `v8platform/ras-grpc-gw@d4b5b77` (2021-09-07)
- Upstream commit: "refactor for access"
- Форк создан: 2025-01-30
- Изменения: 4 коммита в `develop` ветке

### Development

**Коммиты:**
```
1dbeb37 - docs: Add fork documentation
0fbf0db - chore: Upgrade Go from 1.17 to 1.24
b75a481 - feat: Add structured logging and graceful shutdown
a721ca9 - feat: Add HTTP health check endpoints
b96f667 - test: Add comprehensive unit tests
```

**Статистика:**
- +5 новых файлов (logger.go, health.go, 3 test файлов)
- +724 строки тестового кода
- ~400 строк production кода
- Coverage: 97.8% для testable функций

---

## Unreleased

_Планируемые features для следующих версий_

### Planned for v1.1.0-cc

- **Prometheus metrics**
  - `ras_grpc_requests_total` - счётчик запросов
  - `ras_grpc_request_duration_seconds` - латентность
  - `ras_grpc_errors_total` - счётчик ошибок
  - HTTP endpoint `/metrics` для Prometheus scraping

- **Upgrade gRPC** from v1.40.0 → v1.60+
  - Актуальные security patches
  - Новые gRPC features
  - Совместимость с современными клиентами

- **Connection pooling** для RAS connections
  - Пул соединений к RAS серверу
  - Reuse connections для лучшей производительности
  - Configurable pool size

- **Circuit breaker**
  - Защита от cascade failures
  - Automatic recovery
  - Configurable thresholds

### Planned for v1.2.0-cc

- **Configuration management** с `spf13/viper`
  - Поддержка YAML/JSON/ENV конфигурации
  - Environment variables override
  - Config file: `config/config.yaml`

- **Docker образ** с multi-stage build
  - Alpine Linux base
  - Non-root user для безопасности
  - Healthcheck directive
  - Final image < 50 MB

- **Kubernetes manifests**
  - Deployment с resource limits
  - Service (ClusterIP)
  - HorizontalPodAutoscaler
  - ConfigMap и Secret

---

## Maintenance

### Синхронизация с upstream

Процедура описана в `docs/UPSTREAM_SYNC.md`.

**Частота:** Ежемесячно или при critical fixes в upstream.

**Последняя синхронизация:** 2025-01-30 (fork creation)

### Версионирование

Форк использует Semantic Versioning с суффиксом `-cc`:

- `v1.0.0-cc` - Major.Minor.Patch-cc
- Major: Breaking changes
- Minor: New features (backward compatible)
- Patch: Bug fixes

---

**Версия документа:** 1.0
**Последнее обновление:** 2025-01-30
**Maintainer:** defin85
