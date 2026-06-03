# Plan: `mesh-discovery` — discovery aggregator connector

> **Architecture revised (superseding the self-replicating model below).** Instead
> of derived per-datasource providers/connectors + an orchestrator (and the
> mesh-core `DeriveProvider` change), mesh-discovery is now a **single connector**:
> each discovered datasource is emitted as a catalog **`Application`**
> (`ApplicationRef` = datasource id, `Name` = datasource name, `Description` =
> platform), and accounts/groups/roles link to it via the existing
> `ApplicationAccount` / `ApplicationGroup` / `ApplicationRole` edges. The
> orchestrator, `mesh_mgmt_client`, derived-provider options, and per-datasource
> scoping have been removed; a `collect_applications` feature was added. The
> sections below are retained for historical context.

## Context

Hydn's "discovery" system (codename **hydden**) is an aggregator: it crawls many upstream
platforms (Active Directory, Entra, CyberArk, …), each with one or more **datasources**,
and produces a unified inventory of accounts, groups, owners, roles, etc. Today the `control`
backend reads discovery **in-process** (`internal/features/integrations/hydden`) and files
everything under control's own domain (datasources → `Application` records, all accounts pushed
under a single mesh connector).

**Goal:** move sync responsibility *out of `control` and into mesh, where it belongs.* Build a
new `mesh-discovery` connector that runs through the **same execution path as every other mesh
connector** (a subprocess binary mesh-core launches, emitting to the catalog), and organize the
discovered data **under the platform/datasource it came from** instead of one "discovery" bucket.
Over time this connector becomes the system of record for discovery sync and control's in-process
hydden sync is retired (follow-up, not this change).

**Architecture (self-replicating single binary):** one `mesh-discovery` executable run in two
modes via feature options:

1. A base **`mesh-discovery`** provider whose **orchestrator** feature runs the datasource sync
   from hydden and **registers a `mesh-discovery-<platform>` provider + connector per discovered
   datasource** in mesh-core (baking the datasource id into each connector's feature options).
2. Each `mesh-discovery-<platform>` connector runs **the same binary** scoped to one datasource,
   so its collectors emit only that datasource's entities.

Each datasource gets its own provider/connector, so the catalog partitions data by
platform/datasource automatically (`SegmentID = SHA1(connectorID, space)`) — a true aggregator
with no per-entity metadata hacks and no bespoke execution backend.

**Verified feasibility (mesh-core / mesh-sdk):**
- Many providers can share one executable; the binary is the manifest's `"name"`
  (`mesh-core/internal/domain/global/manifests/repositories/manifest_store.go:141-161`).
- Per-connector baked options flow into the subprocess `StartRun`
  (`mesh-core/internal/execution/connector_launcher.go`, `…/on_connector_invocation_requested_dispatch_execution.go:229-246`).
- mesh-core already has provider-create, **connector-create** (`connector_api.go:61-98`), and
  connector-feature-options APIs; control already calls mesh-core's management API with a system
  credential (`control/backend/internal/mesh/client.go` `DoSystem`/`getSystemToken`), and the UC
  registrar already drives provider/feature registration (`…/universal_collector/meshreg/registrar.go`).

**The one mesh-core gap (now in scope):** a provider's `manifestVersion`/`manifestSourceRef`
(which pin the executable) are set only by the internal manifest-import workflow
(`mesh-core/internal/domain/scoped/manifests/workflows/on_manifest_import_requested_reconcile_provider.go:80-91`),
not by the public `PUT provider` API. To let the orchestrator create
`mesh-discovery-<platform>` providers that resolve to the `mesh-discovery` binary, mesh-core needs
a small additive way to register a **derived provider** reusing an existing executable + manifest
version. This is built on a branch as part of this effort.

---

## Deliverables

### A. mesh-core branch — derived-provider registration (additive)
Add a command + endpoint to register a provider for a new `provider_id` that **reuses an existing
executable name and manifest version/source** (the published `mesh-discovery` release), so it
resolves to that binary at invocation time.
- Generalize the existing manifest-import → reconcile flow
  (`providers/commands/reconcile_provider.go:16-67`, the import workflow above) to accept a
  synthesized manifest/feature set for a derived provider id.
- Do **not** touch the subprocess launch path (`connector_launcher.go:391-417`) — only the
  registration side that populates `manifestVersion`/`manifestSourceRef` + feature descriptors.
- Tests: registering a derived provider yields a working executable resolution for its features.

### B. `github.com/hydn-co/mesh-discovery` — new connector repo
Scaffolded from **`mesh-sample`** per
`mesh-utilities/skills/mesh-connector-bootstrap/SKILL.md` (layout, CI/lint/version workflows,
`cmd/main_test.go` smoke tests, `internal/options` split + polymorphic registration,
`connectorutil` helpers, `net/http`-only, `fgrzl/enumerators` pagination, latest `mesh-sdk` tag).

```
mesh-discovery/
├── cmd/main.go                              # manifest + runner.Run()
├── internal/
│   ├── api/discovery_client.go              # minimal hydden read client (auth + paged queries)
│   ├── api/mesh_mgmt_client.go              # minimal mesh-core mgmt client (orchestrator only)
│   ├── credentials/credentials.go
│   ├── mappings/                            # hydden row -> SDK entity transforms
│   ├── options/{core.go,*_options.go,register.go,validate_methods.go}
│   └── collectors/
│       ├── orchestrator/discover_datasources.go
│       ├── account_entity_collector.go
│       ├── group_entity_collector.go
│       ├── owner_entity_collector.go
│       └── application_role_entity_collector.go
```

**Credential:** standard mesh secret, `runner.GrantCredential` (matches hydden's `/auth/api`
`{id, secret}` flow and stays consistent with other connectors). Hydden `base_url` + tenant ref
via feature options. The orchestrator additionally takes a mesh-management credential (standard
secret mechanism, a mesh system grant) + `mesh_core_base_url` option to call the management API.

#### Shared datasource-scope option core (`internal/options/core.go`)
`DatasourceScopeCore { DataSourceID string; Platform string }` embedded by every collector's
options; the orchestrator bakes these per datasource. Collectors **client-side filter** hydden
rows by `DataSourceID` (mirrors `hydden/sync_service.go:696-741`); add server-side `filterModel`
later if hydden supports it.

#### Collector features (datasource-scoped; mirror control's sync, idiomatic mesh folding)
Per the bootstrap skill ("one collector per emitted entity space; fold sub-fetches in"),
membership relationships and per-entity **attribute/detail enrichment** are folded into the
owning entity collector rather than split into separate features:

| Feature | Emits (SDK entities) | hydden source + enrichment | Reference mapping |
|---|---|---|---|
| `collect_accounts` | `Account` (+ profile attributes inline) | `ForEachAccountPage` + `GetAccountDetails` | `mapping_account.go`, `account_attributes_sync_service.go` |
| `collect_groups` | `Group`, `GroupMember` (memberships) | `ForEachGroupPage` + `GetGroupMembers`/`GetGroupDetails` | `mapping_group.go`, `group_attributes_sync_service.go` |
| `collect_owners` | `Person`/`Employee` + owner↔account link | `ForEachOwnerPage` + `ForEachOwnerAccountPage` | `mapping_owner.go` |
| `collect_application_roles` | `Role`/`ApplicationRole`, `AccountRole` (memberships) | `ForEachApplicationRolePage` + `GetAccountAppRoles` | `mapping_app_role.go`, `mapping_app_role_membership.go`, `app_role_membership_sync_service.go` |

Excluded from this build (explicit): **platform users** (`ListPlatformUsers`). **Classifications**
deferred (tagging junction, low priority). **Applications/datasources** are *not* a collector —
they become the provider/connector structure created by the orchestrator.

#### Orchestrator feature `discover_datasources`
- Reads hydden, enumerates distinct datasources — reuse the logic in
  `hydden/sync_service.go` `DiscoverApplications` (`:1133-1252`), keyed on `"Data Source Id"` /
  `"Data Source Name"` / `"Data Source Platform"`.
- For each datasource, via `internal/api/mesh_mgmt_client.go` (deterministic provider id à la
  `meshreg.DeriveProviderID`):
  1. register provider `mesh-discovery-<platform>` reusing the `mesh-discovery` executable
     (deliverable **A**),
  2. `PUT` each collector connector-feature with baked `{data_source_id, platform}` options,
  3. trigger invocations.
- Idempotent: deterministic ids → upserts; re-runs reconcile datasource set.

### C. Account/entity mapping (`internal/mappings`)
Port the field reads + transforms from control's `hydden/mapping_*.go` (do **not** import the
control package — keep the connector self-contained per bootstrap rules). Example account field
map: `"Id"`→`AccountRef` (fallback `"Email"`), `"Account Name"`→`Name`, `"Display Name"`→
`DisplayName`, `"Account Type"`→`AccountType`, `"Email"`→`PrimaryEmail`. `catalog_sync_service.go`
(`mapAccountToCatalogRequest`, `:381-398`) shows today's account→catalog shape.

---

## Assumptions / interpretations
- **"attributes for both roles and groups"** → fold group-detail and role-detail/membership
  enrichment **into** the `collect_groups` and `collect_application_roles` collectors (mesh-idiomatic),
  rather than separate `*_attributes` features as control's DAG has them. Easy to split out if the
  1:1 DAG mapping is preferred.
- **Orchestrator lives in the connector binary**, using a mesh system grant as its secret to call
  the management API — not a control-side worker.
- **Owners** map to `Person`/`Employee` + an account-owner relationship entity (closest SDK shape);
  exact entity choice confirmed during implementation against the SDK catalog.

---

## Build order
0. **Create the `mesh-discovery` repo and commit this plan** as `PLAN.md` (repo root) so the
   generated code can be reviewed against it later.
1. **mesh-core branch (A):** derived-provider registration + tests.
2. **mesh-discovery scaffold + collectors (B/C):** `discovery_client`, options, mappings, the four
   datasource-scoped collectors, with subprocess + entity-contract tests against a fake hydden server.
3. **Orchestrator + mesh_mgmt_client:** wire registration/connector-create/baked-options/invoke;
   end-to-end against a dev mesh-core.
4. **Follow-up (separate change):** deprecate/retire control's in-process hydden sync once parity is confirmed.

## Verification
- mesh-discovery: `go build ./… && go vet ./… && golangci-lint run && go test ./…`; `cmd/main_test.go`
  `testkit.InvokeDescribe`/`InvokeList`; per-collector `entity_contract_test.go` (emitted spaces ==
  `GetSpaces()`); subprocess test feeding `StartRun` with baked datasource options against a fake
  hydden HTTP server, asserting only the scoped datasource's entities are emitted.
- mesh-core: unit tests on derived-provider registration → executable resolution.
- End-to-end: run the base `mesh-discovery` orchestrator against a dev mesh-core pointed at a
  discovery instance; confirm `mesh-discovery-<platform>` providers/connectors appear and each
  connector's catalog holds only its datasource's accounts/groups/owners/roles.

## Key references (reuse, don't reinvent)
- Template & conventions: `mesh-sample/cmd/main.go`, `…/internal/collectors/sample_user_entity_collector.go`, `mesh-utilities/skills/mesh-connector-bootstrap/SKILL.md`.
- Hydden contract, sync DAG & mappings: `control/backend/internal/features/integrations/hydden/{interfaces.go,http_client.go,sync_service.go,mapping_*.go,*_attributes_sync_service.go,catalog_sync_service.go}`.
- mesh management/registration: `…/universal_collector/meshreg/registrar.go`, `control/backend/internal/mesh/client.go`, `mesh-core/internal/web/api/v1/{provider_api.go,connector_api.go}`, `mesh-core/internal/manifests/store.go`, `…/manifests/repositories/manifest_store.go`, the manifest-import workflow + `reconcile_provider.go`.
- SDK: `entities.{Account,Group,GroupMember,Role,AccountRole,Person}`, `spaces`, `runner`, `connector.TypedFeatureContext`, `connectorutil`.
