# mongo2git — Agent Guide

## Overview

**mongo2git** is a Go CLI tool that exports MongoDB collections to version-controlled git repositories. Each document is serialized as a human-readable [MongoDB Extended JSON](https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/) file, committed, and pushed to a remote git repo. Optional Slack notifications report backup results.

```
MongoDB  ──fetch──▶  JSON files  ──commit──▶  Git repo  ──push──▶  Remote
                         │                              │
                         └── one .json per document     └── Slack notification (optional)
                                                      └── Hostname resolved at startup
                                                           and injected into logging + Slack
```

## Architecture

The project follows a clean pipeline pattern with interface-based dependency injection:

```
cmd/mongo2git/main.go       — Entry point: parses config, wires dependencies, runs pipeline
internal/
├── config/config.go        — CLI flags + environment variable parsing
├── mongo/fetcher.go        — MongoDB document fetcher (cursor iteration, O(1) memory)
├── mongo/id.go             — _id extraction → filesystem-safe filename
├── doc/doc.go              — JSON document serialization (Extended JSON)
├── doc/sort.go             — Recursive key-sorting for deterministic output
├── git/repo.go             — Git clone/fetch/reset/commit/push operations
├── hostname/hostname.go    — FQDN resolution (fallback chain)
└── slack/notify.go         — Slack webhook notifications
```

### Interfaces (for testability)

| Interface | File | Methods | Production impl | Mock impl |
|---|---|---|---|---|
| `mongo.Fetcher` | `internal/mongo/fetcher.go` | `ForEach(ctx, fn)` | `MongoFetcher` | `MockFetcher` in `mock.go` |
| `git.RepoOps` | `internal/git/repo.go` | `Prepare()`, `CommitAndPush(timestamp, docCount)` | `GitRepo` | `MockRepo` in `mock.go` |
| `slack.Notifier` | `internal/slack/notify.go` | `Notify(success, timestamp, docCount, detail)` | `SlackNotifier` (enriched with hostname, repo URL, branch) | `MockNotifier` in `mock.go` |

## Key Design Decisions

### 1. Interface-based dependency injection
- `main.go` wires production implementations, tests use mocks
- Mocks live in `mock.go` alongside each package (not a separate `mocks/` dir)
- Any new dependency should follow this pattern: define an interface, implement it, provide a mock

### 2. Deterministic JSON output
- `doc.SortDocument()` recursively converts `bson.M` → `bson.D` with sorted keys
- `doc.WriteDocument` marshals with `MarshalExtJSONIndent` (relaxed mode)
- Same input always produces byte-for-byte identical output — essential for meaningful git diffs

### 3. Filesystem-safe filenames from `_id`
- `mongo.IDToFilename()` handles: `bson.ObjectID` (hex), `bson.Binary` subtype 4 (UUID format), `string` (sanitized - `/` and `\` replaced with `_`), and fallback `fmt.Sprintf`
- Prevents directory traversal from malicious string `_id` values

### 4. Git operations via `os/exec`
- No git library dependency — uses `exec.Command("git", ...)` directly
- `runGit()` helper sets `cmd.Dir`, `cmd.Stdout`, `cmd.Stderr`, `cmd.Env`
- `CommitAndPush` checks `git diff --cached --quiet` exit code to detect changes:
  - exit 0 → no changes, return `false`
  - exit 1 → changes exist, proceed to commit
- `Prepare()` clones if `.git` missing, otherwise fetches + resets + checkouts

### 5. Local mode
- `--local / -L` flag skips git push and Slack HTTP calls (logs instead)
- Useful for testing, CI, or cron without network dependencies

### 6. Configuration: CLI flags + env vars
- `config.ParseFlags()` reads CLI flags (short and long forms), falls back to environment variables
- Optional settings (branch, Slack URL) use `resolveOptional` with a default
- Required settings use `resolve` which errors if neither flag nor env var is set

### 7. Hostname resolution for context-rich logging and notifications
- `hostname.ResolveFQDN()` resolves the machine's fully qualified domain name at startup
- Fallback chain: attempt reverse lookup → fall back to `os.Hostname()` → fall back to `"unknown"`
- The resolved hostname is injected into `slog` as a `"host"` attribute on every log line
- Also passed to `SlackNotifier` so backup notifications include host, repo URL, and branch

## Coding Conventions

### Go style
- `gofmt` enforced (pre-commit hook via lefthook)
- `golangci-lint` with `govet`, `staticcheck`, `errcheck` (pre-commit + CI)
- `go vet` on every commit and CI run
- Module: `github.com/leolvt/mongo2git`, Go 1.26.3

### Package layout
- Each internal package exports: main implementation + interface + mock
- Mocks in `mock.go` within the same package (not a separate test package)
- Tests in `_test.go` files alongside source (not a separate `test/` dir)
- Test files are `package foo` (white-box) not `package foo_test`

### Testing
- `make test` runs `go test -race -cover ./...`
- CI requires all tests to pass before merge
- New functionality must include tests
- Tests should use interfaces/mocks to avoid real MongoDB, git, or HTTP calls
- Integration-style tests (e.g., `git/repo_test.go`) create temp dirs and real git repos (`initTestRepo`)
- Slack tests use `httptest.NewServer`

### Error handling
- Functions return errors to the caller; `main.go` handles top-level errors with `slog.Error` + `os.Exit(1)`
- Warnings and non-fatal issues use `slog.Warn` (e.g., skipping a document with unknown `_id`)
- `main.go` sends a failure Slack notification on fatal error before exiting

### Logging
- Uses `log/slog` with a text handler writing to stderr (source info enabled)
- A `host` attribute (FQDN resolved at startup) is added to every log line via `slog.With("host", host)`
- Info-level: start, commit message, document write, push, backup complete
- Warn-level: disconnect failures, skip documents, Slack send failures
- Error-level: configuration errors, MongoDB failures, git failures, fatal errors

### Naming
- Environment variable names: `UPPER_SNAKE_CASE` (e.g., `MONGO_URI`, `GIT_BRANCH`)
- CLI flags: `--kebab-case` (e.g., `--mongo-uri`, `--git-branch`) with single-letter short forms
- Go identifiers: standard Go camelCase conventions

## Project Structure

```
mongo2git/
├── cmd/mongo2git/main.go       # Entry point
├── internal/
│   ├── config/                 # CLI flags + env var parsing
│   │   ├── config.go
│   │   └── config_test.go
│   ├── doc/                    # Document serialization + key sorting
│   │   ├── doc.go
│   │   ├── sort.go
│   │   ├── doc_test.go
│   │   └── sort_test.go
│   ├── git/                    # Git operations
│   │   ├── repo.go
│   │   ├── mock.go
│   │   └── repo_test.go
│   ├── hostname/               # FQDN resolution
│   │   ├── hostname.go
│   │   └── hostname_test.go
│   ├── mongo/                  # MongoDB fetching + ID extraction
│   │   ├── fetcher.go
│   │   ├── id.go
│   │   ├── mock.go
│   │   ├── id_test.go
│   │   └── fetcher_test.go     # (if added)
│   └── slack/                  # Slack notifications
│       ├── notify.go
│       ├── mock.go
│       └── notify_test.go
├── .github/workflows/
│   ├── ci.yml                  # PR + push CI
│   └── release.yml             # Tag-based release with GoReleaser
├── .goreleaser.yaml            # Release build config (3 OS × 2 arch)
├── Makefile                    # Build, test, lint, release targets
├── lefthook.yml                # Pre-commit + pre-push git hooks
├── mise.toml                   # Tool version management
├── .env.example                # Environment variable template
└── README.md                   # User-facing documentation
```

## CI Pipeline (`.github/workflows/ci.yml`)

Runs on push to `main` and all PRs:
1. `make fmt-check` — formatting
2. `make tidy-check` — module tidiness
3. `make vet` — static analysis
4. `make lint` — golangci-lint
5. `make test` — tests with race detection + coverage
6. `make build` — verify compilation
7. `make release-check` — validate GoReleaser config
8. `make release-snapshot` — validate release build

## Release Process

1. Push a tag matching `v*` (e.g., `git tag v1.0.0 && git push origin v1.0.0`)
2. GitHub Actions runs `release.yml`: CI pipeline + GoReleaser
3. GoReleaser builds for linux/windows/darwin × amd64/arm64, creates archives and publishes to GitHub Releases

## Dependencies (zero runtime deps for production features)

- `go.mongodb.org/mongo-driver/v2` — MongoDB driver (only non-standard-library dependency)
- All other features (git, Slack HTTP, CLI flags, logging) use the Go standard library
- No ORM, no framework, no web server

## Common Tasks for Agents

### Adding a new optional setting
1. Add field to `config.Config`
2. Add CLI flag in `ParseFlags()` using `resolveOptional`
3. Add env var to `.env.example`
4. Wire into `main.go`

### Adding a new command-line flag
1. Add field to `config.Config`
2. Add both long (`--flag-name`) and short (`-X`) flag definitions in `ParseFlags()`
3. Add to the `flag.Usage` custom usage text
4. Add to `Config.Validate()` if validation needed

### Adding a new notification channel
1. Define interface in a new package under `internal/`
2. Provide mock in `mock.go`
3. Wire into `main.go`'s `run()` signature and call sites
4. Update tests in `main_test.go`

### Adding a new MongoDB data transformation
1. Add function in `internal/doc/` or `internal/mongo/`
2. Call it in `run()`'s `ForEach` callback in `main.go`
3. Add tests
