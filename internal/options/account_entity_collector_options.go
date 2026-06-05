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
	// NOTE: spaces.Attributes (the attribute-definition dictionary) is emitted by
	// this collector but intentionally NOT declared here. It is an additive,
	// shared space: declaring it would mark it owned (full-set reconcile, which
	// prunes) and would collide with the group/owner collectors that also emit
	// definitions. Declared spaces are the ones this collector owns/reconciles.
	return []spaces.Space{
		spaces.Accounts,
		spaces.ApplicationAccounts,
		spaces.AccountAttributes,
		spaces.RiskFactors,
		spaces.AccountRiskFactors,
		spaces.Classifications,
		spaces.AccountClassifications,
	}
}

func (o *AccountEntityCollectorOptions) GetRequirements() []string {
	return []string{requirement}
}
