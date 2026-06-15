package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// GroupEntityCollectorOptions configures the group collector (groups,
// memberships, and group<->application links).
type GroupEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
}

func (o *GroupEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/group_entity_collector_options"
}

func (o *GroupEntityCollectorOptions) GetSpaces() []spaces.Space {
	// Per hydn-co/control#1436, group attributes ship as one consolidated
	// GroupExtension per group instead of per-attribute GroupAttribute edges.
	return []spaces.Space{
		spaces.Groups,
		spaces.GroupMembers,
		spaces.ApplicationGroups,
		spaces.GroupExtensions,
	}
}

func (o *GroupEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
