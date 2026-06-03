package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// AccountEntityCollectorOptions configures the account collector.
type AccountEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
	DatasourceScope      `json:",inline"`
}

func (o *AccountEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/account_entity_collector_options"
}

func (o *AccountEntityCollectorOptions) GetSpaces() []spaces.Space {
	return []spaces.Space{spaces.Accounts}
}

func (o *AccountEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
