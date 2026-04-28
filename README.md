# colflow-cli

Collection-flow CLI for Dagster pipelines, data inspection, and ops. Go.

## Build

```sh
go build -o colflow ./cmd/colflow
```

## Install

```sh
go install ./cmd/colflow
```

## Configuration

- `DAGSTER_GRAPHQL_URL` (default `http://127.0.0.1:3000/graphql`)
- `DAGSTER_AUTH` — `user:pass` for HTTP basic auth
- `--url` / `--auth` per-command flags

## Commands

`status`, `runs`, `run`, `logs`, `assets`, `asset`, `graph`, `jobs`, `launch`, `cancel`, `sensors`, `tail`, `reload`, `errors`, `materialise`, `stale`, `config`, `diff`.
