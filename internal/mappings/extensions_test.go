package mappings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

func TestAttributeEntriesFromMapSortsAndMaps(t *testing.T) {
	entries := AttributeEntriesFromMap(map[string]string{
		"Zone":       "us-east",
		"Department": "Eng",
	})

	require.Len(t, entries, 2)
	// Sorted by key for deterministic output.
	assert.Equal(t, "Department", entries[0].Ref)
	assert.Equal(t, "Department", entries[0].Name)
	assert.Equal(t, "Eng", entries[0].Value)
	assert.Empty(t, entries[0].Type, "discovery carries no type hint")
	assert.Equal(t, "Zone", entries[1].Ref)
}

func TestAttributeEntriesFromMapEmpty(t *testing.T) {
	assert.Nil(t, AttributeEntriesFromMap(nil))
	assert.Nil(t, AttributeEntriesFromMap(map[string]string{}))
}

func TestAccountClassificationEntriesProjectsDefsAndEdges(t *testing.T) {
	entries := AccountClassificationEntries(api.Row{
		"Id":              "acc-1",
		"Classifications": "Admin",
		"Is Privileged":   "true",
	})

	refs := make([]string, 0, len(entries))
	for _, e := range entries {
		refs = append(refs, e.Ref)
		assert.Equal(t, 1.0, e.Confidence)
		assert.NotEmpty(t, e.Name)
	}
	assert.ElementsMatch(t, []string{"admin", "privileged"}, refs)
}

func TestAccountClassificationEntriesEmptyWhenNoRef(t *testing.T) {
	assert.Nil(t, AccountClassificationEntries(api.Row{}))
}

func TestAccountRiskFactorEntriesProjectsDefsAndEdges(t *testing.T) {
	entries := AccountRiskFactorEntries(api.Row{
		"Id":                            "acc-1",
		"Accounts with MFA Not Enabled": "5",
	})

	require.Len(t, entries, 1)
	assert.Equal(t, "mfa-not-enabled", entries[0].Ref)
	assert.Equal(t, "Password & Security", entries[0].Category)
	assert.Equal(t, 5.0, entries[0].Weight)
	assert.Equal(t, 1.0, entries[0].Confidence)
	assert.NotEmpty(t, entries[0].Name)
}

func TestAccountRiskFactorEntriesEmptyWhenNoRef(t *testing.T) {
	assert.Nil(t, AccountRiskFactorEntries(api.Row{}))
}
