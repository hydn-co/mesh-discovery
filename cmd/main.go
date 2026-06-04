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

// WithManifest builds the mesh-discovery provider manifest. A single discovery
// connector emits one Application per discovered datasource and links
// accounts/groups/roles to their application via the ApplicationAccount/
// ApplicationGroup/ApplicationRole edges. Every feature needs only the discovery
// base URL (option) plus the Grant credential.
func WithManifest() *runner.Manifest {
	manifest := runner.CreateManifest(
		"mesh-discovery",
		"",
		"Discovery",
		"Mesh aggregator connector for the Hydden discovery platform.",
	)

	manifest.MustRegisterFeature(
		"discovery_application_entity_collector",
		"Collect Applications",
		"Emit one Application per discovered datasource (accounts/groups/roles link to these via edges).",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.ApplicationEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewApplicationEntityCollector),
	)

	manifest.MustRegisterFeature(
		"discovery_account_entity_collector",
		"Collect Accounts",
		"Collect discovery accounts, their full attribute set, and links to their datasource application.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.AccountEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewAccountEntityCollector),
	)

	manifest.MustRegisterFeature(
		"discovery_group_entity_collector",
		"Collect Groups",
		"Collect discovery groups and memberships and link them to their datasource application.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.GroupEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewGroupEntityCollector),
	)

	manifest.MustRegisterFeature(
		"discovery_application_role_entity_collector",
		"Collect Application Roles",
		"Collect discovery application roles (entitlements), role memberships, and datasource links.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.ApplicationRoleEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewApplicationRoleEntityCollector),
	)

	manifest.MustRegisterFeature(
		"discovery_owner_entity_collector",
		"Collect Owners",
		"Collect discovery owners (global identities) as Persons.",
		runner.FeatureSchedulable,
		runner.FeatureTypeCollector,
		new(options.OwnerEntityCollectorOptions),
		(*connector.NoPayload)(nil),
		runner.FeatureResumeBehaviorNone,
		runner.GrantCredential,
		runner.Factory(collectors.NewOwnerEntityCollector),
	)

	if err := manifest.Validate(); err != nil {
		panic(err)
	}
	return manifest
}
