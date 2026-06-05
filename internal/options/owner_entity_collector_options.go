package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// OwnerEntityCollectorOptions configures the owner collector. Owners are global
// identities that transcend datasources, so they are emitted as Persons with no
// application link.
type OwnerEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
}

func (o *OwnerEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/owner_entity_collector_options"
}

func (o *OwnerEntityCollectorOptions) GetSpaces() []spaces.Space {
	return []spaces.Space{spaces.Persons, spaces.PersonAttributes}
}

func (o *OwnerEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
