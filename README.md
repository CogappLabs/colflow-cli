# colflow-cli

CLI for Dagster collection-flow pipelines, parquet inspection, asset scaffolding, and Elasticsearch checks. Go.

## Install (Homebrew)

```sh
brew tap CogappLabs/tap
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

## Where to run colflow

Run from inside a colflow project — anywhere from the project root or any subdirectory works. `colflow` walks up looking for `pyproject.toml`.

| Command | Needs to be inside a project? |
|---------|-------------------------------|
| `inspect`, `sample`, `new-asset`, `start`, `debug` | Yes (uses project root + `output/` + `defs/assets/`) |
| `es-check` | No, but `.env` auto-load only triggers inside a project |
| All Dagster commands (`status`, `runs`, etc.) | No (they hit `DAGSTER_GRAPHQL_URL`) |

Examples:

```sh
cd ~/git/<your-project>
colflow status                       # any project subdir is fine
colflow inspect constituents         # bare name → <root>/output/constituents.parquet
```

If you run a project-aware command outside a project, you'll get `no pyproject.toml found in any parent directory`.

## Project conventions

`inspect`, `sample`, and `new-asset` expect:

- Parquet outputs in `<project-root>/output/`
- Python package at `<project-root>/src/<package>/`
- Asset definitions in `<package>/defs/assets/`
- Tests in `<project-root>/tests/`

A bare name like `colflow inspect constituents` resolves to `<root>/output/constituents.parquet`.

## Commands

### Dagster

All commands accept `--json` for machine-readable output, `--url` / `-u` to override the Dagster URL, and `--auth` / `-a` for HTTP basic auth.

Commands that take an argument (`--id`, `--key`, `--job`) show an interactive picker if the argument is omitted. Enter `0` to cancel.

#### `colflow status` — quick health check

Single-line pipeline health summary.

```sh
colflow status
# Pipeline OK — last run 5m ago (SUCCESS), 16/18 assets materialized

colflow status --json
```

#### `colflow runs` — list recent runs

```sh
colflow runs                    # last 10 runs
colflow runs --limit 20         # last 20 runs
colflow runs --status FAILURE   # only failed runs
colflow runs --json
```

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| `--limit` | `-l` | 10 | Number of runs to show (1-100) |
| `--status` | `-s` | -- | Filter: SUCCESS, FAILURE, STARTED, QUEUED, etc |

#### `colflow run` — run details + logs

```sh
colflow run --id <runId>
colflow run --id <runId> --events 50
```

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| `--id` | -- | (picker) | Run ID |
| `--events` | `-e` | 20 | Number of log events to show |

#### `colflow logs` — filtered log view

```sh
colflow logs --id <runId> --level ERROR
colflow logs --id <runId> --step parse_artists
colflow logs --id <runId> --grep "timed out"
```

| Option | Short | Default | Description |
|--------|-------|---------|-------------|
| `--id` | -- | (picker) | Run ID |
| `--step` | `-s` | -- | Filter by step key |
| `--level` | `-l` | -- | Filter: DEBUG, INFO, WARNING, ERROR |
| `--grep` | `-g` | -- | Filter messages by substring |

#### `colflow errors` — Python tracebacks from a failed run

```sh
colflow errors --id <runId>
```

Extracts step failure events with stack traces. Far quicker than `colflow logs --level ERROR` when you just need the tracebacks.

#### `colflow tail` — live-follow a running job

Polls for new log events and streams them. Exits when the run completes.

```sh
colflow tail --id <runId>
colflow tail --id <runId> --interval 5
```

#### `colflow launch` — start a job run

```sh
colflow launch                      # picker if multiple jobs
colflow launch --job my_pipeline
```

#### `colflow materialise` — re-run specific assets

Bypasses the job system to materialise a subset of assets via `__ASSET_JOB`.

```sh
colflow materialise --asset constituents
colflow materialise --assets "constituents,exhibitions,objects"
```

#### `colflow cancel` — cancel a run

```sh
colflow cancel --id <runId>
```

#### `colflow reload` — reload code location

After editing `defs/`, tell Dagster to pick up changes without restarting `dg dev`.

```sh
colflow reload
```

#### `colflow diff` — compare two runs

Shows step-by-step status diff between runs (which steps now fail/missing/succeed).

```sh
colflow diff --run1 <id-a> --run2 <id-b>
```

### Assets

#### `colflow assets` — list all assets

```sh
colflow assets
colflow assets --json
```

#### `colflow asset` — detailed single asset

Group, compute kind, dependencies, staleness, recent materialisations, kinds, tags, freshness.

```sh
colflow asset --key constituents
```

#### `colflow graph` — asset dependency graph

```sh
colflow graph
```

#### `colflow stale` — list stale assets

Filters to assets that need re-materialisation, with stale causes.

```sh
colflow stale
```

#### `colflow config` — run config schema for a job

```sh
colflow config --job famsf_pipeline
```

### Sensors and jobs

#### `colflow sensors` — sensor status + recent ticks

```sh
colflow sensors
```

#### `colflow jobs` — list jobs

```sh
colflow jobs
```

### Dev server

#### `colflow start` — start the Dagster dev server

Wraps `uv run dg dev` from the project root.

```sh
colflow start
```

#### `colflow debug` — start the Dagster dev server with debugpy

Wraps `DAGSTER_DEBUG=1 uv run dg dev`. Listens for an IDE debugger on port 5678.

```sh
colflow debug
```

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

Examples:

```sh
# Use ELASTICSEARCH_URL + ELASTICSEARCH_API_KEY from .env automatically
colflow es-check

# Check a specific index
colflow es-check collection_documents

# Pull from a non-default env var (single-quote so the shell doesn't expand $)
colflow es-check --url '$ELASTICO_URL' --api-key '$ELASTICO_API_KEY'

# List all indices, JSON output
colflow es-check --indices --json
```

### Scaffolding

- `colflow new-asset [name]` — generate Dagster asset (Polars `pl.LazyFrame` + Pandera schema + asset_check) and a test stub. With no name, runs interactively:
  - pulls live asset list from Dagster GraphQL (or `defs/assets/` filenames if Dagster is down)
  - numbered or name-based upstream picker (multi-select via comma)
  - default group suggested from most-common existing group

Templates align with collection-flow Commandments: every asset has a `description=`, every schema has `class Config: name = "..."`, and `group=extract` adds `kinds={"http"}` + `retry_policy=api_retry_policy`.

Flags: `--upstream=a,b`, `--group=name`, `--title="..."`, `--test=false`, `--dry-run`.

## colflow vs dg — when to use which

`colflow` and `dg` (Dagster's official CLI) overlap on launching runs but serve different purposes.

| Task | `colflow` | `dg` |
|------|-----------|------|
| Start dev server | `colflow start` (wraps `uv run dg dev`) | `dg dev` |
| Scaffold assets | `colflow new-asset` | `dg scaffold` |
| Validate definitions | -- | `dg check defs` |
| Launch a job | `colflow launch --job X` | `dg launch --job X` |
| Launch specific assets | `colflow materialise --assets X,Y` | `dg launch --assets X,Y` |
| List recent runs | `colflow runs` | -- |
| View run logs / filter errors | `colflow logs --level ERROR` | -- |
| Tracebacks from a failure | `colflow errors --id X` | -- |
| Live-follow a running job | `colflow tail --id X` | -- |
| Cancel a run | `colflow cancel --id X` | -- |
| List assets | `colflow assets` | -- |
| Asset detail (deps, staleness) | `colflow asset --key X` | -- |
| Stale assets only | `colflow stale` | -- |
| Asset dependency graph | `colflow graph` | -- |
| Compare two runs step-by-step | `colflow diff --run1 X --run2 Y` | -- |
| Reload code location | `colflow reload` | -- |
| Sensor health | `colflow sensors` | -- |
| Inspect a parquet output | `colflow inspect <name>` | -- |
| Sample / filter parquet rows | `colflow sample <name> --where ...` | -- |
| Test ES connection | `colflow es-check` | -- |
| Pipeline health summary | `colflow status` | -- |
| JSON output for scripting/Claude | `colflow X --json` | -- |

**Use `dg`** for project management: scaffolding (when `colflow new-asset` doesn't fit), validating definitions.

**Use `colflow`** for runtime observability and the day-to-day loop: what's running, why it failed, what's stale, what's in the parquet outputs, and is the search index alive.

## Common debugging workflows

```sh
# Quick health check
colflow status

# Find the latest failure and inspect it
colflow runs --status FAILURE --limit 1
colflow errors --id <runId>

# Why an asset is stale
colflow asset --key <name>

# Re-run a single asset and watch
colflow materialise --asset <name>
colflow tail --id <runId>

# Cancel a stuck run
colflow cancel --id <runId>

# After editing defs/, pick up changes without restarting dg dev
colflow reload

# Inspect what an asset actually wrote
colflow inspect <name>
colflow sample <name> -n 5 --where status=published

# Verify the search cluster is healthy
colflow es-check
colflow es-check collection_documents
```

## LLM-friendly output

`--json` is the supported format for Claude / scripting. Every command emits structured JSON with no ANSI codes.

```sh
colflow status --json | jq '.latest.status'
colflow runs --json | jq '.[0].runId'
colflow assets --json | jq '[.[] | select(.assetMaterializations | length == 0)] | length'
colflow logs --id <runId> --level ERROR --json | jq '.[].message'
colflow inspect <name> --json | jq '.dagster.stale_status'
```

## Colour coding

Run statuses are colour-coded in terminal output:

| Colour | Statuses |
|--------|----------|
| Green | SUCCESS |
| Red | FAILURE |
| Yellow | STARTED, STARTING |
| Blue | QUEUED |
| Gray | CANCELED, CANCELING, NOT_STARTED |

## Dagster GraphQL API usage

The CLI queries the standard Dagster GraphQL API:

| Query/Mutation | Used by | Description |
|----------------|---------|-------------|
| `runsOrError` | `runs`, `status` | List runs with optional status filter |
| `runOrError` | `run`, `logs`, `tail`, `errors`, `diff` | Run details, step stats, event log |
| `assetNodes` | `assets`, `graph`, `stale`, `inspect` | All assets with materialisations / deps |
| `assetNodeOrError` | `asset`, `inspect` | Single asset with full metadata |
| `runConfigSchemaOrError` | `config` | Job config schema |
| `sensorsOrError` | `sensors` | Sensor state and recent ticks |
| `repositoryOrError` | `jobs` | List jobs in the repository |
| `repositoriesOrError` | (internal) | Auto-discover repo name/location |
| `launchRun` | `launch`, `materialise` | Mutation: start a run |
| `terminateRun` | `cancel` | Mutation: cancel a running job |
| `reloadRepositoryLocation` | `reload` | Mutation: reload code location |

## Releasing

Tag pushes to `v*` build cross-platform binaries via GoReleaser and update the Homebrew tap.

```sh
git tag v0.2.3
git push origin v0.2.3
```

The action builds darwin/linux × amd64/arm64 archives, publishes a GitHub release, and updates the formula in `CogappLabs/homebrew-tap`.

Repo secret `TAP_GITHUB_TOKEN` (PAT with `contents:write` on `CogappLabs/homebrew-tap`) is required for the formula push step.

The `--version` output is wired to ldflags-injected `main.version`, `main.commit`, `main.date`.
