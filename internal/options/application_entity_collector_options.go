package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// ApplicationEntityCollectorOptions configures the application collector, which
// emits one Application per discovered datasource.
type ApplicationEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
}

func (o *ApplicationEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/application_entity_collector_options"
}

func (o *ApplicationEntityCollectorOptions) GetSpaces() []spaces.Space {
	return []spaces.Space{spaces.Applications}
}

func (o *ApplicationEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
