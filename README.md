# mesh-discovery

A mesh aggregator connector for the **Hydden discovery** platform.

Discovery is itself an aggregator: it crawls many upstream platforms (Active
Directory, Entra, CyberArk, …), each with one or more **datasources**, and
produces a unified inventory of accounts, groups, owners, and roles.
`mesh-discovery` reads that inventory and emits it to the mesh catalog, organized
**under the platform/datasource it came from**.

## Self-replicating aggregator model

A single `mesh-discovery` binary runs in two modes via feature options:

1. The base **`mesh-discovery`** provider hosts the collectors below plus an
   orchestrator that enumerates discovery datasources and registers a
   **`mesh-discovery-<platform>` provider + connector per datasource** in
   mesh-core — each reusing this same binary, with a `data_source_id` baked into
   its options.
2. Each per-datasource connector emits only that datasource's entities, so the
   catalog partitions data by platform/datasource automatically.

See [PLAN.md](PLAN.md) for the full design and rationale.

## Features

| Feature | Emits | Scope |
|---|---|---|
| `collect_accounts` | `Account` | per datasource (by `data_source_id`) |
| `collect_groups` | `Group`, `GroupMember` | per datasource (by `data_source_name`) |
| `collect_application_roles` | `Role`, `AccountRole` | per datasource |
| `collect_owners` | `Person` | global (base provider only) |

Credentials use the standard mesh **Grant credential**
(`{client_id, client_secret}`) for the discovery `/auth/api` flow. The discovery
base URL is a feature option (`base_url`).

## Build & test

```bash
go build ./...
go test ./...
go vet ./...
golangci-lint run
```
