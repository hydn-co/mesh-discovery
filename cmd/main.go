package main

import (
	"github.com/hydn-co/mesh-sdk/pkg/connector"
	"github.com/hydn-co/mesh-sdk/pkg/runner"

	"github.com/hydn-co/mesh-discovery/internal/collectors"
	"github.com/hydn-co/mesh-discovery/internal/options"
)

func main() {
	runner.Run(WithManifest())
}

// WithManifest builds the mesh-discovery provider manifest.
//
// The base mesh-discovery provider hosts the collector features below. The
// orchestrator (added separately) reuses this same binary under dynamically
// registered mesh-discovery-<platform> providers, baking a data_source_id into
// each connector's options so each emits only its datasource's entities.
func WithManifest() *runner.Manifest {
	manifest := runner.CreateManifest(
		"mesh-discovery",
		"",
		"Discovery",
		"Mesh aggregator connector for the Hydden discovery platform.",
	)

	manifest.MustRegisterFeature(
		"collect_accounts",
		"Collect Accounts",
		"Collect discovery accounts, optionally scoped to a single datasource.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.AccountEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewAccountEntityCollector),
	)

	manifest.MustRegisterFeature(
		"collect_groups",
		"Collect Groups",
		"Collect discovery groups and memberships, optionally scoped to a single datasource.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.GroupEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewGroupEntityCollector),
	)

	manifest.MustRegisterFeature(
		"collect_application_roles",
		"Collect Application Roles",
		"Collect discovery application roles (entitlements) and per-account role memberships.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.ApplicationRoleEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewApplicationRoleEntityCollector),
	)

	manifest.MustRegisterFeature(
		"collect_owners",
		"Collect Owners",
		"Collect discovery owners (global identities) as Persons. Registered on the base provider only.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.OwnerEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewOwnerEntityCollector),
	)

	manifest.MustRegisterFeature(
		"discover_datasources",
		"Discover Datasources",
		"Enumerate discovery datasources and register a mesh-discovery-<platform> provider + connector per datasource in mesh-core.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.DiscoverDatasourcesOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewDiscoverDatasourcesCollector),
	)

	if err := manifest.Validate(); err != nil {
		panic(err)
	}
	return manifest
}
