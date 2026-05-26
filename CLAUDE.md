# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build ./cmd/bumblebee          # build the binary
go test ./...                     # run all tests
go test -race ./...               # run with race detector (required for CI)
go test ./internal/scanner/...    # run a single package's tests
go vet ./...                      # vet
gofmt -l .                        # check formatting (should print nothing)
./bumblebee selftest              # end-to-end smoke test against embedded fixtures
```

Build with a version stamp:
```sh
go build -ldflags "-X main.Version=v0.1.1" -o bumblebee ./cmd/bumblebee
```

CI runs gofmt, go vet, go test -race, go build, and bumblebee selftest on both ubuntu-latest and macos-latest, plus govulncheck separately.

## Architecture

### Data flow

`cmd/bumblebee` parses flags → resolves filesystem roots (profile-dependent) → `scanner.Run` walks roots and dispatches recognized filenames to ecosystem parsers → `output.Emitter` deduplicates and writes NDJSON records to the sink → `exposure.Catalog.MatchAll` turns matched package records into `finding` records → `scan_summary` terminates the run.

### Package layout

- **`cmd/bumblebee/`** — CLI dispatch and all flag handling. `main.go` owns `scan`/`roots` subcommands; `selftest.go` embeds fixture files for the smoke test; `roots.go` resolves per-profile scan roots; `sink.go` constructs the output destination (stdout / file / HTTP).

- **`internal/model/`** — The wire schema: `Record` (packages), `Finding` (catalog hits), `ScanSummary`, `Diagnostic`. Each type exposes a `StableID()` that hashes a canonical identity tuple with SHA-256 — this is the `record_id` on every emitted record, stable across runs. Do not change the tuple fields in any `StableID()` without bumping `SchemaVersion` and adding a new `docs/schema/vX.Y.Z/` directory.

- **`internal/scanner/`** — Orchestrates one scan. Spawns `cfg.Concurrency` worker goroutines reading from a `jobs` channel. The walker callback pattern-matches filenames and sends typed jobs (`"npm-lock"`, `"py-dist"`, etc.) to workers. Workers call the appropriate ecosystem scanner method. `rootKindFor()` stamps each record with the `RootKind` of its enclosing configured root.

- **`internal/walk/`** — Filesystem walker wrapping `filepath.WalkDir`. Enforces exclude lists (by basename and suffix path component), skips symlinked directories, and tracks visited inodes to prevent loops. `DefaultExcludes` is a large curated list; operators can extend it with `--exclude`.

- **`internal/exposure/`** — Loads and matches against operator-supplied JSON catalogs. Matching is exact `(ecosystem, normalized_name, version)`. `Load()` accepts a file or a directory of `*.json` files (merged, must share `schema_version`). `MatchAll()` returns all matching entries so one package/version pair matched by multiple catalog files produces multiple findings.

- **`internal/output/`** — `Emitter` wraps two `json.Encoder` instances (records sink and stderr diagnostics). `ObservePackage` deduplicates within a run via the `seen` map keyed on `record_id`; findings are not separately deduped — they rely on package dedup. The records sink can optionally implement `StatsReporter` to surface HTTP delivery counters into `scan_summary`.

- **`internal/normalize/`** — `PyPI()` applies PEP 503 (lowercase, collapse `-`/`_`/`.` runs to `-`). `NPM()` lowercases. Used both when emitting records and when indexing catalog entries so catalog package names written naturally still match.

- **`internal/ecosystem/`** — One subdirectory per ecosystem: `npm`, `pnpm`, `yarn`, `bun`, `pypi`, `gomod`, `rubygems`, `composer`, `mcp`, `editorext`, `browserext`. Each exposes a `Scanner` struct initialized with `MaxFileSize`, `Emit func(model.Record)`, and `Diag func(level, path, msg string)`. Scanners are single-threaded per file; the orchestrator owns concurrency.

- **`internal/endpoint/`** — Reads hostname, OS, arch, username, and UID from the runtime environment.

### Schema versioning

`model.SchemaVersion = "0.1.0"` is stamped on every record. A breaking wire-format change requires:
1. New directory `docs/schema/vX.Y.Z/` with updated JSON Schema files.
2. Bump `model.SchemaVersion`.
3. Do not edit a published schema in place.

### Exposure catalogs

Maintained under `threat_intel/`. Each file must have `schema_version`, `entries`, and a `_comment` root field. Each entry needs `id`, `ecosystem`, `package`, `versions`, and a `source` URL. The only severity value currently in use is `"critical"`.

### Commit style

Conventional commits: `fix(scope): ...`, `feat(scope): ...`, `docs: ...`, `ci: ...`. Keep PRs small; separate refactors from behaviour changes. Update `README.md` for any user-facing flag, profile, ecosystem, or output field change.
