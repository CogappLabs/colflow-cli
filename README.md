# colflow-cli

CLI for Dagster collection-flow pipelines, parquet inspection, asset scaffolding, and Elasticsearch checks. Go.

## Install (Homebrew)

The release binaries live on the public `CogappLabs/colflow-cli` repo. The Homebrew tap is private, so cloning it needs SSH access to `CogappLabs/homebrew-tap`.

```sh
brew tap CogappLabs/tap git@github.com:CogappLabs/homebrew-tap.git
brew install colflow
```

Update later with `brew upgrade colflow`.

## Install (from source)

```sh
go install github.com/CogappLabs/colflow-cli/cmd/colflow@latest
```

## Build locally

```sh
go build -o colflow ./cmd/colflow
```

## Configuration

### Dagster

- `DAGSTER_GRAPHQL_URL` (default `http://127.0.0.1:3000/graphql`)
- `DAGSTER_AUTH` — `user:pass` for HTTP basic auth
- `--url` / `--auth` per-command flags

### Elasticsearch

- `ELASTICSEARCH_URL` (default `http://localhost:9200`)
- `ELASTICSEARCH_API_KEY` (optional)
- `--url` / `--api-key` flags accept `'$VAR'` to read from any env var

### `.env` auto-load

When a command runs inside a colflow project (parent directory has `pyproject.toml`), `<project-root>/.env` and `.env.local` are loaded automatically. Existing OS env vars take precedence. Combined with `'$VAR'` flag syntax: `colflow es-check --url '$ELASTICSEARCH_URL'` works without manually exporting.

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

### Elasticsearch

- `colflow es-check [index]` — verify cluster reachability. Reports cluster name + status (or `serverless (reachable)` for Elastic Cloud Serverless). With an index argument, also reports its health, doc count, and store size.
  - `--url` / `--api-key` accept plain values or `'$VAR'` to read from env / `.env`.
  - `--indices` lists all indices via `/_cat/indices`.
  - `--insecure` skips TLS verification.
  - `--json` for structured output.
  - Pretty error output with hints for common failure modes (401/403/404/429/503, DNS, connection-refused, TLS, timeout).

### Scaffolding

- `colflow new-asset [name]` — generate Dagster asset (Polars `pl.LazyFrame` + Pandera schema + asset_check) and a test stub. With no name, runs interactively:
  - pulls live asset list from Dagster GraphQL (or `defs/assets/` filenames if Dagster is down)
  - numbered or name-based upstream picker (multi-select via comma)
  - default group suggested from most-common existing group

Templates align with collection-flow Commandments: every asset has a `description=`, every schema has `class Config: name = "..."`, and `group=extract` adds `kinds={"http"}` + `retry_policy=api_retry_policy`.

Flags: `--upstream=a,b`, `--group=name`, `--title="..."`, `--test=false`, `--dry-run`.

## LLM-friendly output

`--json` is the supported format for Claude / scripting. Every command emits structured JSON with no ANSI codes.

## Releasing

Tag pushes to `v*` build cross-platform binaries via GoReleaser and update the Homebrew tap.

```sh
git tag v0.2.3
git push origin v0.2.3
```

The action builds darwin/linux × amd64/arm64 archives, publishes a GitHub release, and updates the formula in `CogappLabs/homebrew-tap`.

Repo secret `TAP_GITHUB_TOKEN` (PAT with `contents:write` on `CogappLabs/homebrew-tap`) is required for the formula push step.

The `--version` output is wired to ldflags-injected `main.version`, `main.commit`, `main.date`.
