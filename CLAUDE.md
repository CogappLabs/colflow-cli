# colflow-cli

Go CLI for Dagster collection-flow pipelines. Wraps Dagster GraphQL, reads Parquet outputs, scaffolds assets, checks Elasticsearch.

## Architecture

- `cmd/colflow/main.go` — root urfave/cli command, registers all subcommands. Calls `project.LoadDotEnv` at startup.
- `internal/client/` — Dagster GraphQL client. `client.go` is transport, `queries.go` per-operation wrappers, `types.go` response shapes. All Dagster API access goes through `Query()`.
- `internal/format/` — terminal output helpers (colour, `TimeAgo`, `FormatTimestamp`, `PadRight`). `PadRight` is ANSI-aware.
- `internal/prompts/` — interactive numbered pickers (`Pick`, `SelectRun`, `SelectAsset`, `SelectJob`) and free-text prompts (`Ask`, `Confirm`). Shared `bufio.Reader` so piped stdin flows through successive prompts.
- `internal/project/` — detect project root via `pyproject.toml`, derive package name + `output/` + `defs/assets/` paths. `LoadDotEnv` calls godotenv on `<root>/.env` and `.env.local`. Resolves bare asset names to `<root>/output/<name>.parquet`.
- `internal/commands/` — one file per command, each returning a `*cli.Command` (urfave/cli/v3). `common.go` exposes `CommonFlags()` (`--url`, `--auth`, `--json`) + `ApplyCommon` and `PrintJSON` helpers. `treeprint.go` builds the Parquet schema tree (collapses list/map wrappers). `escheck.go` is the Elasticsearch helper. `devserver.go` wraps `uv run dg dev` for `start` / `debug` (foreground, inherits stdio). `templates/` holds embedded `.tmpl` files used by `new-asset` (text/template).

## Conventions

- urfave/cli/v3 for CLI. Validate arg counts in the `Action` via `c.NArg()` checks.
- All Dagster commands accept `--url`, `--auth`, `--json`. Append `commands.CommonFlags()...` to per-command flag slice; call `commands.ApplyCommon(c)` first thing in `Action`.
- Inspect/sample/new-asset auto-detect the Python project by walking up from cwd. They use `project.Detect(project.Cwd())`.
- Bare-name resolution for parquet args: literal path → `<root>/output/<arg>` → `<root>/output/<arg>.parquet`.
- Picker tags use `format.Green("[asset]")` etc. Dagster lookup is best-effort; commands degrade gracefully when Dagster is down.
- `printDagsterSection` in `inspect.go` is silent on Dagster failure or missing asset — by design.
- `'$VAR'` syntax in es-check flags reads from process env (which includes loaded `.env`). Implemented in `resolveEnvRef`.
- Pretty errors over raw payloads. `ESError` parses JSON; `printESError` adds context-specific hints.
- **Never** scaffold `automation_condition=when_all_deps_updated` in new-asset templates. Deliberate per-asset choice, not a default.

## Adding a new command

1. Create `internal/commands/<name>.go` returning `*cli.Command`.
2. If it talks to Dagster: build flag slice with `append(CommonFlags(), <extra flags>...)`; call `ApplyCommon(c)` at start of `Action`.
3. For Parquet args: call `resolveOrPick(c.Args().Slice())` (defined in `inspect.go`) for consistent picker behaviour.
4. Register in `cmd/colflow/main.go`.
5. Build: `go build -o colflow ./cmd/colflow`.
6. After adding: update both `README.md` (per-command section + vs-dg table) and `CLAUDE.md` (this file's architecture map if file structure changed).

## Available commands

`status`, `runs`, `run`, `logs`, `errors`, `tail`, `launch`, `materialise`, `cancel`, `reload`, `diff`, `assets`, `asset`, `graph`, `stale`, `config`, `sensors`, `jobs`, `inspect`, `sample`, `new-asset`, `es-check`, `start`, `debug`. (24 commands.)

## GraphQL schema notes

- `metadataEntries` is a union — use typed inline fragments (`IntMetadataEntry`, `TextMetadataEntry`, `PathMetadataEntry`, `JsonMetadataEntry`, `BoolMetadataEntry`, `FloatMetadataEntry`).
- `sensorsOrError` requires a `repositorySelector`.
- Asset materialisation events come through `ExecutionStepFailureEvent`, `MessageEvent` etc. — Dagster's `DagsterRunEvent` is a union with no direct fields.
- Repository auto-discovered via `repositoriesOrError` — never hardcode names.
- Dagster timestamps are sometimes seconds, sometimes ms strings. `format.tsToSeconds` heuristic: `>1e12` → ms.

## Parquet handling

- Pure Go via `github.com/parquet-go/parquet-go` — no CGO, no shellout to duckdb.
- `FlattenSchema` walks the schema tree and collapses Parquet list/map encodings (`foo.list.element.bar` → list[group] with `bar` indented; `foo.key_value.{key,value}` → `map[k -> v]`).
- `collapseLeafPath` produces matching labels for null-count rows (`foo[].bar`, `foo{}.bar`).
- `computeNulls` reads each column chunk's pages and sums `NumNulls()`. Avoid for huge files — paginated read is full-scan.
- `sample --where field=value` reads in 64-row batches until N matches collected or `--max-scan` hit.

## Elasticsearch handling

- `esGet` parses error JSON into `ESError{Status, Type, Reason}`. Cobra error reprint suppressed via `os.Exit(1)` after rendering.
- Hints in `hintForESError` cover 401/403/404/429/503 + DNS/refused/TLS/timeout transport errors.
- Elastic Cloud Serverless returns 410 on `/_cluster/health`. Fallback to `GET /` which exists everywhere.
- Index check via `/_cat/indices/<name>?format=json&bytes=b`.

## Templating (new-asset)

- `internal/commands/templates/asset.py.tmpl` and `test.py.tmpl` are loaded via `//go:embed`.
- Use Go `text/template` syntax (`{{.Name}}`, `{{if .IsExtract}}…{{end}}`, `{{range .Upstream}}…{{end}}`).
- `assetData` struct fields drive everything — extend the struct + the template, not strings.ReplaceAll.

## Release

GoReleaser on `v*` tag → GitHub release with darwin/linux × amd64/arm64 binaries → updates `CogappLabs/homebrew-tap` formula.

Token: `TAP_GITHUB_TOKEN` repo secret (PAT with `contents:write` on the tap repo).

`--version` reads `main.version`, `main.commit`, `main.date` injected via ldflags. Don't hardcode the version in `cmd/colflow/main.go`.

`brews:` block in `.goreleaser.yaml` is officially deprecated upstream in favour of `homebrew_casks:`. Casks need notarised binaries on macOS, which we don't currently sign — switching is a bigger project. Stay on `brews` until that's tackled.

## Project status

Public repo (`CogappLabs/colflow-cli`) — release tarballs anonymously downloadable. Public Homebrew tap (`CogappLabs/homebrew-tap`) — `brew tap CogappLabs/tap && brew install colflow` is enough.
