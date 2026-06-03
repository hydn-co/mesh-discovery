# Agent instructions — mesh-discovery

## Before any task
- Read this file and [PLAN.md](PLAN.md) first.
- Read the collector you are modifying (or the closest one) before changing code.
- If you touch capability registration or option schemas, read `cmd/main.go`,
  `internal/options/core.go`, `internal/options/register.go`, and the relevant
  feature option file first.
- The discovery API field names and contract live in control's hydden package
  (`control/backend/internal/features/integrations/hydden/`); treat it as the
  source of truth when changing `internal/api` or `internal/mappings`.

## Project summary
- **mesh-discovery**: a mesh aggregator connector for the Hydden discovery
  platform. Discovery aggregates many platforms/datasources; this connector
  re-emits them to the mesh catalog partitioned by datasource via a
  self-replicating provider model (see PLAN.md).
- **Entry point**: `cmd/main.go`
- **Framework**: [mesh-sdk](https://github.com/hydn-co/mesh-sdk) (pinned in `go.mod`)
- **Language**: Go (see `go.mod`)

## Non-negotiable rules
1. Collectors embed `*connector.TypedFeatureContext` and implement `Init`/`Start`/`Stop`.
2. Factory functions take `*connector.TypedFeatureContext` and return `runner.Feature`.
3. Options layout: shared cores in `internal/options/core.go`, one file per
   feature, registration in `register.go`, `Validate()` in `validate_methods.go`.
4. Use `mesh-sdk/pkg/connectorutil` for logging/validation/credential extraction;
   do not recreate those helpers.
5. All discovery API calls go through `internal/api` using `net/http` only — no
   third-party HTTP clients or provider SDKs.
6. Collectors build their client via the `newClient` seam so tests can inject a
   fake (see `internal/collectors/entity_contract_test.go`).
7. Datasource scoping: accounts scope by `data_source_id`; groups and roles scope
   by `data_source_name` (group/role rows carry name, not id); memberships scope
   via the in-scope group/account set.
8. Wrap errors with `fmt.Errorf("context: %w", err)`; check `ctx.Err()` in loops.
9. Behavioral test names: `TestShould{Expectation}When{Condition}`.

## Primary sources
| Need | Source |
|------|--------|
| Collector examples | `internal/collectors/*.go` |
| Options | `internal/options/` |
| Mappings | `internal/mappings/` |
| Discovery API client | `internal/api/discovery_client.go` |
| Manifest | `cmd/main.go` |
| Design | `PLAN.md` |
