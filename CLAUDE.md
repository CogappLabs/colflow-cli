# colflow-cli

Go CLI for Dagster collection-flow pipelines. Wraps Dagster GraphQL + reads Parquet outputs.

## Architecture

- `cmd/colflow/main.go` — root cobra command, registers all subcommands.
- `internal/client/` — Dagster GraphQL client. `client.go` is transport, `queries.go` per-operation wrappers, `types.go` response shapes. All Dagster API access goes through `Query()`.
- `internal/format/` — terminal output helpers (colour, `TimeAgo`, `FormatTimestamp`, `PadRight`). `PadRight` is ANSI-aware.
- `internal/prompts/` — interactive numbered pickers (`Pick`, `SelectRun`, `SelectAsset`, `SelectJob`) and free-text prompts (`Ask`, `Confirm`). Shared `bufio.Reader` so piped stdin flows through successive prompts.
- `internal/project/` — detect project root via `pyproject.toml`, derive package name + `output/` + `defs/assets/` paths. Resolves bare asset names to `<root>/output/<name>.parquet`.
- `internal/commands/` — one file per command, each returning a `*cobra.Command`. `common.go` has `CommonFlags` (`--url`, `--auth`, `--json`) and `PrintJSON` helper. `treeprint.go` builds the Parquet schema tree (collapses list/map wrappers).

## Conventions

- Cobra for CLI, no zod/validator — Cobra's `cobra.MaximumNArgs` etc. do the validation.
- All Dagster commands accept `--url`, `--auth`, `--json`. Add via `commands.AddCommon`.
- Inspect/sample/new-asset auto-detect the Python project by walking up from cwd. They use `project.Detect(project.Cwd())`.
- Bare-name resolution for parquet args: literal path → `<root>/output/<arg>` → `<root>/output/<arg>.parquet`.
- Picker tags use `format.Green("[asset]")` etc. Dagster lookup is best-effort; commands degrade gracefully when Dagster is down.
- `printDagsterSection` in `inspect.go` is silent on Dagster failure or missing asset — by design.
- All asset templates live as Go string consts in `newasset.go`. Imports include `from collection_flow.support.polars.validation import validate_dataframe` — that helper is required.

## Adding a new command

1. Create `internal/commands/<name>.go` returning `*cobra.Command`.
2. Use `CommonFlags` + `AddCommon` if it talks to Dagster.
3. For Parquet args: call `resolveOrPick(args)` (defined in `inspect.go`) for consistent picker behaviour.
4. Register in `cmd/colflow/main.go`.
5. Build: `go build -o colflow ./cmd/colflow`.

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

## Release

GoReleaser on `v*` tag → GitHub release with darwin/linux × amd64/arm64 binaries → updates `CogappLabs/homebrew-tap` formula.

Token: `TAP_GITHUB_TOKEN` repo secret (PAT with `contents:write` on the tap repo).

## Project status

Private repo (`CogappLabs/colflow-cli`). Distribution via private Homebrew tap (`CogappLabs/homebrew-tap`).
