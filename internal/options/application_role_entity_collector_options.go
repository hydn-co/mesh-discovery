package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// ApplicationRoleEntityCollectorOptions configures the application-role collector
// (roles, role<->application links, and account<->role memberships).
type ApplicationRoleEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
}

func (o *ApplicationRoleEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/application_role_entity_collector_options"
}

func (o *ApplicationRoleEntityCollectorOptions) GetSpaces() []spaces.Space {
	return []spaces.Space{spaces.Roles, spaces.AccountRoles, spaces.ApplicationRoles}
}

func (o *ApplicationRoleEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
