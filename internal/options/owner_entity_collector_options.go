package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// OwnerEntityCollectorOptions configures the owner collector. Owners are global
// identities that transcend datasources, so this collector is NOT datasource
// scoped — the orchestrator registers it on the base mesh-discovery provider
// only, where it emits the full owner/identity catalog as Persons.
type OwnerEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
}

func (o *OwnerEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/owner_entity_collector_options"
}

func (o *OwnerEntityCollectorOptions) GetSpaces() []spaces.Space {
	return []spaces.Space{spaces.Persons}
}

func (o *OwnerEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
