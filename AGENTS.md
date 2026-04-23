# Repository Guidelines

## Project Structure & Module Organization

Dujiao-Next API is a Go backend service. The executable entry point is `cmd/server/main.go`. Application wiring lives in `internal/app` and `internal/provider`; routing and middleware are in `internal/router`; business logic is in `internal/service`; persistence is split between `internal/models` and `internal/repository`. Shared DTOs, config, logging, queue, cache, payment, and worker code live in matching `internal/*` packages. Tests are colocated as `*_test.go`. Runtime configuration starts from `config.yml.example`.

## Build, Test, and Development Commands

- `go mod tidy`: normalize module dependencies.
- `go run cmd/server/main.go`: run the API locally; health check is `GET /health`.
- `go test ./...`: run all package tests.
- `go test -cover ./...`: run tests with coverage output.
- `go fmt ./...`: format Go source before committing.
- `goreleaser release --clean --snapshot`: verify release packaging locally without publishing.

If local services are enabled, ensure Redis and the selected database are available. SQLite is the default example database; PostgreSQL-specific tests may require PostgreSQL.

## Coding Style & Naming Conventions

Use standard Go formatting and idioms. Keep package names short and lowercase, and name files by feature, for example `payment_service.go` or `product_repository.go`. Preserve service/repository boundaries: router-facing packages handle HTTP concerns, while domain workflows belong in `internal/service`. Do not hardcode secrets, tokens, payment credentials, or production DSNs; read them from configuration.

## Testing Guidelines

Use Go's built-in testing package. Place tests beside the code they cover and name them `TestXxx` in `*_test.go` files. Add focused unit tests for service, DTO, crypto, and helper logic; add repository tests when database behavior changes. Run `go test ./...` before opening a pull request, and `go test -cover ./...` for shared business logic.

## Commit & Pull Request Guidelines

Recent history uses concise Chinese prefixes such as `修复：`, `优化：`, and `新增：`; keep that style or use conventional equivalents like `fix:`, `feat:`, and `chore:`. Keep commits scoped to one logical change. Pull requests should include a short problem summary, the implemented approach, test results, linked issues when applicable, and screenshots or API examples for behavior visible to users or admins.

## Security & Configuration Tips

Copy `config.yml.example` for local setup and replace all placeholder secrets before production use. Never commit local config files, database files, logs, private keys, credentials, or generated payment certificates. Validate external callbacks and webhooks carefully, and prefer parameterized GORM queries over raw SQL string concatenation.
