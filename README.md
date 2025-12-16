# Go Base Boilerplate

Production-ready Go service starter kit with authentication, JWT (RS256), Redis caching, RabbitMQ pub/sub, and PostgreSQL via GORM.

## Prerequisites

- Go 1.22+
- PostgreSQL 14+
- Redis 6+
- RabbitMQ 3.12+
- RSA key pair for JWT signing (`configs/keys/private.pem`, `configs/keys/public.pem`)

## Configuration

All configuration lives in `configs/config.yaml` and can be overridden via environment variables prefixed with `GOBASE_`. Key sections:

- `database.dsn` – PostgreSQL connection string.
- `redis.*` – Redis host, password, DB, and default TTL.
- `rabbitmq.*` – Broker URI, exchange, queue, routing key.
- `jwt.*` – Paths to RSA keys and token metadata.
- `seed.*` – Enables initial admin seed data.

Example DSN for a local Dockerized Postgres instance:

```
postgres://postgres:postgres@localhost:5432/go_base?sslmode=disable
```

Ensure the referenced database, user, and password exist before starting the app.

## CLI Commands

Commands are registered through Cobra in `cmd/root.go`; subcommands are defined under `cmd/`. Currently available commands:

- `serve` – Runs the HTTP API server.
- `seed` – Seeds initial data (admin user) when enabled in config.
- `generate crud` – Produces DTOs, validation-ready requests, repositories, services, controllers, and JWT-protected routes from an existing GORM model.

Main entrypoint is `main.go`, which imports `cmd` and invokes `cmd.Execute()`.

### Running without installation

```
go run ./main.go serve
```

### Building the binary

```
go build -o bin/go-base ./main.go
./bin/go-base serve
```

Add `bin/` to your PATH or install globally with `go install ./...` to expose `go-base` as a shell command.

## Environment Setup

1. **PostgreSQL**: create user & database matching your DSN. Example:
   ```bash
   psql -U postgres -c "CREATE ROLE gobase LOGIN PASSWORD 'gobase';"
   createdb -U gobase go_base
   ```
2. **Redis**: ensure Redis is running and reachable at the configured address.
3. **RabbitMQ**: ensure the configured exchange/queue exist or allow auto-declare with provided credentials.
4. **JWT keys**: place RSA private/public keys under `configs/keys/` or update config paths accordingly.
5. **Seed admin** (optional): toggle `seed.enabled` and set `seed.admin_email` / `seed.admin_password`.

## Database Schema

Schema objects are managed through GORM's auto-migration in `internal/database/migrations/migrate.go`, where each model to be synchronized is registered. By default, only `internal/modules/user/model.go` is included.

To apply the schema automatically at startup, enable the flag in `configs/config.yaml`:

```yaml
database:
  auto_migrate: true
```

Or export the environment variable before running the service:

```bash
export GOBASE_DATABASE_AUTO_MIGRATE=true
go run ./main.go serve
```

On launch, the `serve` command will invoke the migration routine before the HTTP server starts. Add additional models to the list in `internal/database/migrations/migrate.go` to have their tables managed alongside the existing ones.

### Adding new GORM models

1. **Define the model** – Create your struct inside the appropriate package (e.g., `internal/modules/order/model.go`) and annotate it with GORM tags as needed (see `internal/modules/user/model.go` for reference).
2. **Register it for migrations** – Import the package inside `internal/database/migrations/migrate.go` and add the struct pointer to the `db.AutoMigrate(...)` list, for example:

   ```go
   return db.AutoMigrate(
       &user.User{},
       &order.Order{},
   )
   ```

3. **Enable auto-migrate when running** – Set `database.auto_migrate: true` in `configs/config.yaml` or export `GOBASE_DATABASE_AUTO_MIGRATE=true` before executing `go run ./main.go serve`. The `serve` command triggers the migration routine on startup (`cmd/serve.go`).

After these steps, launching the service will prompt GORM to create or evolve the underlying tables automatically. For seed data, the `cmd/seed` command can be used once the schema exists.

For more advanced schema changes, you can use the `Migrator` API directly within the same file, for example:

```go
if err := db.Migrator().CreateConstraint(&MyModel{}, "fk_my_model_other"); err != nil {
    return err
}
```

Keep these operations idempotent—GORM will execute them on every startup when `auto_migrate` is true.

## CRUD Generator

Scaffold CRUD modules directly from a GORM model:

```
go run ./main.go generate crud --model internal/book/model.go --struct Book
```

Or after building the binary:

```
./bin/go-base generate crud --model internal/book/model.go --struct Book
```

The generator expects:

1. An existing model file (`--model`) containing the target struct.
2. An optional `--struct` override when the file defines multiple structs; defaults to the first struct found.

Generated files (`*_gen.go`) are placed next to the model and include:

- DTOs for create/update plus response mappers with `validation.Validator` tags.
- Repository/service layers wired for GORM, including typed ID parsing.
- Gin controller with DTO validation, JWT auth middleware, and RESTful CRUD routes.
- Module bootstrap that self-registers with the HTTP router via `internal/httpserver/modules`.

If any of the target files already exist, generation aborts to avoid overwriting manual changes. Delete or relocate previous outputs before re-running.

## Project Structure

See `docs/architecture.md` for a detailed layout of packages and responsibilities.

## Development Tips

- Run `go fmt ./...` and `go test ./...` regularly.
- Configure environment variables for local overrides (e.g., `GOBASE_DATABASE_DSN`).
- Use Docker Compose to spin up Postgres, Redis, and RabbitMQ for local development (not included yet).
