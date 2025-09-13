# Repository Guidelines

## Project Structure & Modules
- `cmd/vkftpd/`: CLI entrypoint (Cobra) and config loading.
- `pkg/`: Core packages — `authentication/`, `authorization/`, `ftpserver/`, `lpc/`, `logging/`, `users/`.
- `docs/`: Protocol and integration notes (LPC format, auth, access tree).
- `resources/`: Sample data used by docs/tests.
- `.github/workflows/ci.yml`: Go 1.22 CI for test, build, release.

## Build, Test, Develop
- `make build` — builds `vkftpd` embedding git version.
- `go build ./cmd/vkftpd` — manual build.
- `./vkftpd --version` — show version; `./vkftpd --config config.json` — run locally.
- `go test ./...` — run all tests; `-race` for race checks; `-v` for verbose; `go test -v ./pkg/authentication/...` for a package.
- Optional sanity: `go vet ./...` before pushing.

## Coding Style & Naming
- Go ≥ 1.22. Format with `gofmt -s -w .` (CI expects idiomatic Go).
- Package names lowercase, no underscores; exported identifiers in `CamelCase`, unexported in `camelCase`.
- Keep files small and cohesive; prefer table-driven functions and clear error wrapping (`fmt.Errorf("...: %w", err)`).
- Add GoDoc comments for exported types/functions, especially in `pkg/`.

## Testing Guidelines
- Use `testing` and `testify/assert` (already in `go.mod`).
- Place tests in `*_test.go`; name `TestXxx(t *testing.T)`; use table-driven tests.
- Prefer in-memory fakes/mocks (see `users` and `authorization` tests). Guard slow/real file tests with `testing.Short()`.
- Ensure new code has unit tests and keeps `go test ./...` green with `-race`.

## Commit & PR Guidelines
- Prefer Conventional Commits (e.g., `feat: ...`, `fix: ...`); reference issues/PRs (e.g., `(#5)`).
- PRs should: describe the change and rationale, link issues, include tests, and update docs/README if behavior or config changes.
- Keep PRs focused; ensure CI passes.

## Security & Configuration
- Do not commit real configs, credentials, or TLS keys. Use examples in README.
- For FTPS, verify cert/key paths; for PASV mode, ensure `pasv_port_range` is open and `pasv_address` correct.
- Avoid logging secrets; use structured logging helpers in `pkg/logging`.

## Agent-Specific Tips
- Make minimal, targeted changes; preserve public APIs.
- Follow module boundaries in `pkg/`; update or add tests adjacent to changes.
