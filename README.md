# colflow-cli

CLI for Dagster collection-flow pipelines, parquet inspection, and asset scaffolding. Go.

## Install (Homebrew)

```sh
brew tap CogappLabs/tap git@github.com:CogappLabs/homebrew-tap.git
brew install colflow
```

Update: `brew upgrade colflow`.

## Install (from source)

```sh
go install github.com/CogappLabs/colflow-cli/cmd/colflow@latest
```

Requires `GOPRIVATE=github.com/CogappLabs` for the private repo:

```sh
go env -w GOPRIVATE=github.com/CogappLabs
```

## Build locally

```sh
go build -o colflow ./cmd/colflow
```

## Configuration

- `DAGSTER_GRAPHQL_URL` (default `http://127.0.0.1:3000/graphql`)
- `DAGSTER_AUTH` — `user:pass` for HTTP basic auth
- `--url` / `--auth` per-command flags

## Project conventions

`inspect`, `sample`, and `new-asset` auto-detect the project by walking up from the current directory looking for `pyproject.toml`. They expect:

- Parquet outputs in `<project-root>/output/`
- Python package at `<project-root>/src/<package>/`
- Asset definitions in `<package>/defs/assets/`
- Tests in `<project-root>/tests/`

A bare name like `colflow inspect constituents` resolves to `<root>/output/constituents.parquet`.

## Commands

### Dagster

`status`, `runs`, `run`, `logs`, `assets`, `asset`, `graph`, `jobs`, `launch`, `cancel`, `sensors`, `tail`, `reload`, `errors`, `materialise`, `stale`, `config`, `diff`.

All support `--json` for scripting / LLM consumption.

### Data

- `colflow inspect [file | asset_name]` — schema tree (Parquet list/map collapsed), size, rows, populated %, plus Dagster metadata when the asset is known: group, kinds, stale status + causes, last materialisation datetime, upstream/downstream deps.
- `colflow sample [file | asset_name] -n 5 [--where field=value]` — pretty-print rows, or `--json`. Repeatable `--where` filters by equality on a (possibly nested) field path.

No-arg form lists `output/` with tags: `[asset]` (matches Dagster), `[orphan]` (no match).

### Scaffolding

- `colflow new-asset [name]` — generate Dagster asset (Polars `pl.LazyFrame` + Pandera schema + asset_check) and a test stub. With no name, runs interactively:
  - pulls live asset list from Dagster GraphQL (or `defs/assets/` filenames if Dagster is down)
  - numbered or name-based upstream picker (multi-select via comma)
  - default group suggested from most-common existing group

Flags: `--upstream=a,b`, `--group=name`, `--title="..."`, `--test=false`, `--dry-run`.

## LLM-friendly output

`--json` is the supported format for Claude / scripting. Every command emits structured JSON with no ANSI codes.

## Releasing

Tag pushes to `v*` build cross-platform binaries via GoReleaser and update the Homebrew tap.

```sh
git tag v0.1.0
git push origin v0.1.0
```

Repo secret `TAP_GITHUB_TOKEN` (PAT with `contents:write` on `CogappLabs/homebrew-tap`) is required.
