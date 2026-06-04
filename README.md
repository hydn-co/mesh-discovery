# mesh-discovery

A mesh aggregator connector for the **Hydden discovery** platform.

Discovery is itself an aggregator: it crawls many upstream platforms (Active
Directory, Entra, CyberArk, …), each with one or more **datasources**, and
produces a unified inventory of accounts, groups, owners, and roles.
`mesh-discovery` reads that inventory and emits it to the mesh catalog, organized
**under the platform/datasource it came from**.

## Model

A single **`mesh-discovery`** connector represents every discovered datasource
as a catalog **`Application`** (`ApplicationRef` = datasource id, `Name` =
datasource name, `Description` = platform). Accounts, groups, and roles are
emitted normally and linked to their datasource application through the existing
catalog **edge** entities (`ApplicationAccount`, `ApplicationGroup`,
`ApplicationRole`). Accounts carry the datasource id directly; groups and roles
carry the datasource name, which is resolved to an id via an index built from
the account feed (mirroring how control resolves `ApplicationID` by name).

See [PLAN.md](PLAN.md) for the design and rationale.

## Features

| Feature | Emits |
|---|---|
| `discovery_application_entity_collector` | `Application` (one per datasource) |
| `discovery_account_entity_collector` | `Account`, `ApplicationAccount`, `Attribute`, `AccountAttribute` |
| `discovery_group_entity_collector` | `Group`, `GroupMember`, `ApplicationGroup` |
| `discovery_application_role_entity_collector` | `Role`, `AccountRole`, `ApplicationRole` |
| `discovery_owner_entity_collector` | `Person` |

Account **attributes** are collected by streaming each datasource's full entity
records via `/internal/v1/datastore/fetch` (the connector probes the datasource's
entity-type variants, then flattens each record): distinct attribute names become
`Attribute` entities and per-account values are stored on `AccountAttribute`
edges. Group attributes are deferred — the SDK has no `GroupAttribute` edge yet.

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
