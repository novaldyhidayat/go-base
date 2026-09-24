# Architecture Overview

This project follows a modular, layered architecture to encourage separation of concerns and maintainability.

```
cmd/
  root.go
  serve.go
  migrate.go
  seed.go
  generate.go
  generate_crud.go
configs/
  config.yaml
internal/
  app/
    application.go
    buildinfo/
      buildinfo.go
  modules/
    auth/
      controller.go
      dto.go
      module.go
      service.go
  cache/
    redis.go
  database/
    postgres.go
  httpserver/
    server.go
    middleware/
      auth.go
      requestid.go
      recovery.go
      cors.go
  logger/
    logger.go
  mq/
    rabbitmq.go
  security/
    jwt.go
    password.go
  seed/
    seed.go
  modules/user/
    model.go
    repository.go
    service.go
  validation/
    validator.go
pkg/
  response/
    response.go
```

## Layers

1. **Transport Layer**: `internal/httpserver` provides HTTP server setup with Gin, middleware, and routing. Controllers live in feature packages such as `auth` and `user`, exposing handlers.
2. **Service Layer**: Business logic resides in services under feature packages (e.g., `auth/service.go`, `user/service.go`). Services orchestrate repositories, caching, and message queue operations.
3. **Repository Layer**: Repository structs encapsulate data persistence with GORM. Each domain package (`auth`, `user`) implements its own repository with model definitions under `user/model.go`.
4. **Infrastructure Layer**: Shared infrastructure clients and utilities (database, cache, message queue, logging, JWT) live under `internal/database`, `internal/cache`, `internal/mq`, `internal/logger`, and `internal/security`.
5. **Common Utilities**: Shared response helpers and utilities reside under `pkg/` to keep reusable components outside domain-specific packages.

## Configuration & Environment

- Configuration files are stored under `configs/` and read via Viper. Environment variables override file values using the `GOBASE_` prefix.
- RSA keys for JWT signing are referenced from `configs/config.yaml`; keep private keys out of version control and replace development keys in production.
- Build metadata (version, commit) is injected via linker flags and exposed through `internal/app/buildinfo`.

## Commands

- `go-base serve`: Runs the HTTP server with graceful shutdown and health checks.
- `go-base migrate`: Applies database migrations (GORM auto-migrate).
- `go-base seed`: Runs data seeders conditionally based on configuration.
- `go-base generate crud`: Generates boilerplate CRUD code skeleton for a specified entity.

## Validation Strategy

- DTO-level validation uses `go-playground/validator` via the `internal/validation` adapter.
- Domain-specific validation hooks can be added per entity (e.g., `user/validation.go`).

## Security Considerations

- Password hashing uses `bcrypt` via `golang.org/x/crypto`.
- JWT tokens use RS256 signing with per-request validation middleware.
- Configurable CORS middleware and request ID generation are provided by default.

## Observability

- Structured logging is provided through `zap` with an environment-aware setup (development vs production).
- Request/response metrics and tracing hooks can be integrated in `internal/httpserver/middleware` as future enhancements.

## Pub/Sub and Caching

- RabbitMQ connection pool is established in `internal/mq` with publisher/subscriber helpers.
- Redis client is initialized in `internal/cache` with functions for caching services.

## Testing Strategy

- Unit tests focus on services and repositories using mock interfaces.
- Integration tests can leverage Docker Compose with PostgreSQL, Redis, and RabbitMQ.
