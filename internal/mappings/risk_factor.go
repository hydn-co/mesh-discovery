package mappings

import (
	"strconv"
	"strings"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// riskFactorDef describes one discovery risk indicator and how it maps to a
// catalog RiskFactor. gsKey is the discovery account-grid column; Hydden emits a
// non-empty (numeric weight) cell when the risk applies to an account — matching
// the boolIndicator fields in control's mapping_account.go.
type riskFactorDef struct {
	gsKey    string
	ref      string
	name     string
	category string
}

// accountRiskFactorDefs is the catalog of account risk factors discovery
// reports. Categories follow Hydden's own risk buckets (the "(Total)"
// groupings). See docs/discovery-field-mapping.md.
var accountRiskFactorDefs = []riskFactorDef{
	{"Accounts with MFA Not Enabled", "mfa-not-enabled", "MFA Not Enabled", "Password & Security"},
	{"Accounts with Password 90+ Days", "password-age-90", "Password Age 90+ Days", "Password & Security"},
	{"Accounts with Password 180+ Days", "password-age-180", "Password Age 180+ Days", "Password & Security"},
	{"Accounts with Password 365+ Days", "password-age-365", "Password Age 365+ Days", "Password & Security"},
	{"Accounts with Password Never Set", "password-never-set", "Password Never Set", "Password & Security"},
	{
		"Account Password Not Changed Since Public Breach",
		"password-not-changed-since-breach",
		"Password Not Changed Since Public Breach",
		"Password & Security",
	},
	{"Accounts not used in 90+ Days", "stale-90", "Not Used in 90+ Days", "Account Activity"},
	{"Accounts not used in 180+ Days", "stale-180", "Not Used in 180+ Days", "Account Activity"},
	{"Accounts not used in 365+ Days", "stale-365", "Not Used in 365+ Days", "Account Activity"},
	{
		"Accounts with 10+ Failed Login Attempts in 1 Hour",
		"failed-logins",
		"10+ Failed Login Attempts in 1 Hour",
		"Account Activity",
	},
	{"Shared Accounts", "shared-account", "Shared Account", "Account Statistics"},
	{"Account Group Deviation", "group-deviation", "Account Group Deviation", "Group Membership"},
	{"Accounts with No Owner", "no-owner", "No Owner", "Owner Mapping"},
	{
		"Inactive Owners With Enabled Accounts",
		"inactive-owner-enabled",
		"Inactive Owner With Enabled Account",
		"Owner Mapping",
	},
	{"Privileged Accounts Not Vaulted", "privileged-not-vaulted", "Privileged Account Not Vaulted", "Privilege"},
	{"Breached Account(s)", "breached-account", "Breached Account", "Breach Data"},
}

// AccountRiskFactors returns the RiskFactor definitions and AccountRiskFactor
// edges implied by an account row. A risk factor applies when its grid cell is
// non-empty (Hydden sends a numeric weight); that weight is captured on the
// definition. Confidence is always 1.0 — the risk is asserted by discovery.
func AccountRiskFactors(row api.Row) ([]*entities.RiskFactor, []*entities.AccountRiskFactor) {
	accountRef := AccountRef(row)
	if accountRef == "" {
		return nil, nil
	}

	var (
		defs  []*entities.RiskFactor
		edges []*entities.AccountRiskFactor
	)
	for _, d := range accountRiskFactorDefs {
		cell := getString(row, d.gsKey)
		if cell == "" {
			continue
		}
		defs = append(defs, &entities.RiskFactor{
			Metadata:      types.EntityMetadata{Space: spaces.RiskFactors},
			RiskFactorRef: d.ref,
			Name:          d.name,
			Category:      d.category,
			Weight:        parseWeight(cell),
		})
		edges = append(edges, &entities.AccountRiskFactor{
			Metadata:      types.EntityMetadata{Space: spaces.AccountRiskFactors},
			AccountRef:    accountRef,
			RiskFactorRef: d.ref,
			Confidence:    1,
		})
	}
	return defs, edges
}

// parseWeight reads the numeric severity Hydden ships in a risk-indicator cell;
// non-numeric cells contribute no weight.
func parseWeight(cell string) float64 {
	if f, err := strconv.ParseFloat(strings.TrimSpace(cell), 64); err == nil {
		return f
	}
	return 0
}

// riskFactorGSKeys is the set of grid columns consumed as risk factors, so the
// attribute mapper can skip them.
func riskFactorGSKeys() map[string]struct{} {
	out := make(map[string]struct{}, len(accountRiskFactorDefs))
	for _, d := range accountRiskFactorDefs {
		out[d.gsKey] = struct{}{}
	}
	return out
}
