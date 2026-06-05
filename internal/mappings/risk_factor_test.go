package mappings

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

func TestShouldEmitRiskFactorPerNonEmptyIndicator(t *testing.T) {
	row := api.Row{
		"Id":                             "acc-1",
		"Accounts with MFA Not Enabled":  "5",
		"Accounts not used in 365+ Days": "10",
		"Shared Accounts":                "", // empty -> not a risk
	}

	defs, edges := AccountRiskFactors(row)
	require.Len(t, defs, 2)
	require.Len(t, edges, 2)

	byRef := map[string]float64{}
	for _, d := range defs {
		assert.Equal(t, spaces.RiskFactors, d.Metadata.Space)
		assert.NotEmpty(t, d.Name)
		assert.NotEmpty(t, d.Category)
		byRef[d.RiskFactorRef] = d.Weight
	}
	assert.Equal(t, 5.0, byRef["mfa-not-enabled"])
	assert.Equal(t, 10.0, byRef["stale-365"])

	for _, e := range edges {
		assert.Equal(t, spaces.AccountRiskFactors, e.Metadata.Space)
		assert.Equal(t, "acc-1", e.AccountRef)
		assert.Equal(t, 1.0, e.Confidence, "discovery-sourced risk is asserted at full confidence")
	}
}

func TestShouldCategorizeRiskFactorsByHyddenBucket(t *testing.T) {
	row := api.Row{
		"Id":                              "acc-1",
		"Accounts with MFA Not Enabled":   "1",
		"Privileged Accounts Not Vaulted": "1",
		"Breached Account(s)":             "1",
		"Account Group Deviation":         "1",
	}
	defs, _ := AccountRiskFactors(row)
	cat := map[string]string{}
	for _, d := range defs {
		cat[d.RiskFactorRef] = d.Category
	}
	assert.Equal(t, "Password & Security", cat["mfa-not-enabled"])
	assert.Equal(t, "Privilege", cat["privileged-not-vaulted"])
	assert.Equal(t, "Breach Data", cat["breached-account"])
	assert.Equal(t, "Group Membership", cat["group-deviation"])
}

func TestShouldReturnNoRiskFactorsWhenNoneApply(t *testing.T) {
	defs, edges := AccountRiskFactors(api.Row{"Id": "acc-1"})
	assert.Empty(t, defs)
	assert.Empty(t, edges)
}

func TestShouldReturnNilRiskFactorsWhenNoAccountRef(t *testing.T) {
	defs, edges := AccountRiskFactors(api.Row{"Accounts with MFA Not Enabled": "1"})
	assert.Nil(t, defs)
	assert.Nil(t, edges)
}

func TestShouldParseRiskWeight(t *testing.T) {
	assert.Equal(t, 7.0, parseWeight("7"))
	assert.Equal(t, 2.5, parseWeight(" 2.5 "))
	assert.Equal(t, 0.0, parseWeight("high"))
	assert.Equal(t, 0.0, parseWeight(""))
}

func TestRiskFactorGSKeysCoverCatalog(t *testing.T) {
	keys := riskFactorGSKeys()
	assert.Len(t, keys, len(accountRiskFactorDefs))
	_, ok := keys["Accounts with MFA Not Enabled"]
	assert.True(t, ok)
}
