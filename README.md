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

## Commands

### Dagster

`status`, `runs`, `run`, `logs`, `assets`, `asset`, `graph`, `jobs`, `launch`, `cancel`, `sensors`, `tail`, `reload`, `errors`, `materialise`, `stale`, `config`, `diff`.

### Data

- `colflow inspect <file.parquet>` — schema, size, row count, null %.
- `colflow sample <file.parquet> -n 5` — pretty-print rows (or `--json`).

### Scaffolding

- `colflow new-asset <name> [--upstream=a,b] [--group=transform]` — generate Dagster asset (Polars + Pandera + asset_check) and test stub.

## Releasing

Tag pushes to `v*` build cross-platform binaries via GoReleaser and update the Homebrew tap.

```sh
git tag v0.1.0
git push origin v0.1.0
```

A repo secret `TAP_GITHUB_TOKEN` (PAT with `contents:write` on `CogappLabs/homebrew-tap`) is required.
