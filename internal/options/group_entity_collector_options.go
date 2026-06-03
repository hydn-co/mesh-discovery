package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// GroupEntityCollectorOptions configures the group collector (groups + memberships).
type GroupEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
	DatasourceScope      `json:",inline"`
}

func (o *GroupEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/group_entity_collector_options"
}

func (o *GroupEntityCollectorOptions) GetSpaces() []spaces.Space {
	return []spaces.Space{spaces.Groups, spaces.GroupMembers}
}

func (o *GroupEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
