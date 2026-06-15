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
	// Per hydn-co/control#1436, person attributes ship as one consolidated
	// PeopleExtension per person instead of per-attribute PersonAttribute edges.
	return []spaces.Space{spaces.Persons, spaces.PeopleExtensions}
}

func (o *OwnerEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
