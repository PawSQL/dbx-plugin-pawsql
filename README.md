# PawSQL Optimizer for DBX

A [DBX](https://github.com/t8y2/dbx) plugin that runs [PawSQL](https://www.pawsql.com) SQL
optimization on any pasted SQL statement and renders a structured report (rule violations,
recommended index DDLs, SQL rewrites, and a What-If validation report).

It is the DBX counterpart of the PawSQL DBeaver/JetBrains plugin, reduced to its core
optimization flow: **paste SQL → optimize → structured report**, with server / anonymous
configuration and a database-type selector. Support performance validation if you supply a online data source on PawSQL server, or run schema-only analysis via `dbType`.

## Architecture

```
[DBX workbench HTML/JS]  --(sidecar protocol v1, stdio-jsonl)-->  [Go sidecar]  --(HTTPS JSON)-->  PawSQL API
```

The sandboxed plugin iframe cannot open sockets (its CSP forbids it), so every byte of
network traffic is performed by the Go sidecar, which the workbench drives over the DBX
sidecar protocol (JSON-RPC 2.0 over JSON Lines). The sidecar owns the PawSQL API contract
and orchestration (`createAnalysisAsync` → poll `getAnalysisSummary` → `getStatementDetails`)
and returns a normalized report; the frontend only renders it.

## Layout

```
manifest.json            DBX plugin manifest (id/publisher/version/permissions/entrypoints)
dbx-plugin.toml          packaging config for @dbx-app/plugin-cli
.dbx-store.json          first-submission store metadata (optional)
assets/plugin.png       icon (DBeaver paw print)
ui/                      workbench (index.html, app.css, i18n.js, util.js, bridge.js, report.js, app.js)
backend/                 Go sidecar (main.go, pawsql.go, config.go, pawsql_test.go)
.github/workflows/       release workflow
```

## Building

Prerequisites: Go 1.22+, Node 20+.

```bash
# Run the Go unit tests (httptest, no network)
cd backend && go test ./... && cd ..

# Build one .dbxp (also cross-compiles the sidecar for 5 platforms)
npx @dbx-app/plugin-cli package .
```

The package step produces `dist/com.pawsql.dbx-1.0.0-<target>.dbxp` per platform and an
`dist/*.artifact.json` metadata file (sha256 + size) used to fill the store candidate.

## Local development

```bash
npx @dbx-app/plugin-cli dev
```

## Publishing to the DBX store

1. Push this repository to a public GitHub repo (update `source` / `homepage` in
   `manifest.json`, `.dbx-store.json`, and the store candidate to that repo).
2. Create a `v1.0.0` release; the workflow builds and attaches the `.dbxp` artifacts.
3. Compute each artifact's `sha256` / `size` (the release workflow's `*.artifact.json`
   already contains them) and fill the `targets` array in
   `dbx-store\candidates\com.pawsql.dbx.json`.
4. Open a single PR to `t8y2/dbx-store` with `publishers/pawsql.json` and
   `candidates/com.pawsql.dbx.json`, then ask a maintainer to run the signing workflow.

See `dbx-store\CONTRIBUTING.md` for the full submission rules.

## Configuration

| Setting | Default | Notes |
| --- | --- | --- |
| Server URL | `https://www.pawsql.com` | PawSQL API base URL |
| Auth mode | anonymous | Anonymous uses a shared key; login stores only the returned user key (the password is never persisted) |
| Database type | MySQL | One of the supported PawSQL types (`mysql`, `postgres`, `oracle`, …) |
| Validate / Analyze / Rewrite / Dedup index / Index only / Update stats | on | Maps to the `createAnalysisAsync` flags |

## License

Apache-2.0. See `LICENSE`; third-party notices in `NOTICE`.
