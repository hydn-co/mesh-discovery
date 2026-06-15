package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// AccountEntityCollectorOptions configures the account collector.
type AccountEntityCollectorOptions struct {
	DiscoveryOptionsCore `json:",inline"`
}

func (o *AccountEntityCollectorOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/account_entity_collector_options"
}

func (o *AccountEntityCollectorOptions) GetSpaces() []spaces.Space {
	// Per hydn-co/control#1436, this collector emits one consolidated
	// AccountExtension per account (attributes, classifications, and risk factors
	// inline) instead of the per-edge fan-out into the attribute/classification/
	// risk-factor definition and edge spaces.
	return []spaces.Space{
		spaces.Accounts,
		spaces.ApplicationAccounts,
		spaces.AccountExtensions,
	}
}

func (o *AccountEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
