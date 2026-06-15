package collectors

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/options"
)

func runAccountCollector(t *testing.T) []any {
	t.Helper()
	emitter := &captureEntityEmitter{}
	c := &AccountEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.AccountEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)
	return emitter.emitted
}

// findAccountExtension returns the single AccountExtension for ref, failing if
// it is missing or duplicated (one consolidated entity per account is the #1436
// invariant).
func findAccountExtension(t *testing.T, emitted []any, ref string) *entities.AccountExtension {
	t.Helper()
	var found *entities.AccountExtension
	for _, e := range emitted {
		if ext, ok := e.(*entities.AccountExtension); ok && ext.AccountRef == ref {
			require.Nil(t, found, "account %s must have exactly one AccountExtension", ref)
			found = ext
		}
	}
	require.NotNil(t, found, "expected an AccountExtension for %s", ref)
	return found
}

// assertNoLegacyCatalogEntities guards the consolidation: none of the old
// per-edge / definition entity types may be emitted any longer.
func assertNoLegacyCatalogEntities(t *testing.T, emitted []any) {
	t.Helper()
	for _, e := range emitted {
		switch e.(type) {
		case *entities.Attribute,
			*entities.AccountAttribute,
			*entities.GroupAttribute,
			*entities.PersonAttribute,
			*entities.Classification,
			*entities.AccountClassification,
			*entities.RiskFactor,
			*entities.AccountRiskFactor:
			t.Errorf("legacy fan-out entity %T must no longer be emitted", e)
		}
	}
}

func TestShouldFoldAccountRiskFactorsIntoExtensionAtFullConfidence(t *testing.T) {
	emitted := runAccountCollector(t)
	ext := findAccountExtension(t, emitted, "acc-1")

	require.Len(t, ext.RiskFactors, 1, "acc-1 has exactly one risk indicator in the fixture")
	assert.Equal(t, "mfa-not-enabled", ext.RiskFactors[0].Ref)
	assert.Equal(t, "Password & Security", ext.RiskFactors[0].Category)
	assert.Equal(t, 5.0, ext.RiskFactors[0].Weight)
	assert.Equal(t, 1.0, ext.RiskFactors[0].Confidence)

	assertNoLegacyCatalogEntities(t, emitted)
}

func TestShouldFoldAccountClassificationsIntoExtensionAtFullConfidence(t *testing.T) {
	emitted := runAccountCollector(t)
	ext := findAccountExtension(t, emitted, "acc-1")

	var admin *entities.ClassificationEntry
	for i := range ext.Classifications {
		if ext.Classifications[i].Ref == "admin" {
			admin = &ext.Classifications[i]
		}
	}
	require.NotNil(t, admin, "Classifications=\"Admin\" yields the admin classification")
	assert.Equal(t, 1.0, admin.Confidence)
}

func TestShouldFoldAccountAttributesIntoExtension(t *testing.T) {
	emitted := runAccountCollector(t)
	ext := findAccountExtension(t, emitted, "acc-1")

	assert.NotEmpty(t, ext.Attributes, "the account extension carries the account's attributes inline")
	assertNoLegacyCatalogEntities(t, emitted)
}

func TestShouldEmitOneAccountExtensionPerAccount(t *testing.T) {
	emitted := runAccountCollector(t)

	accounts := map[string]struct{}{}
	extensions := map[string]int{}
	for _, e := range emitted {
		switch v := e.(type) {
		case *entities.Account:
			accounts[v.AccountRef] = struct{}{}
		case *entities.AccountExtension:
			extensions[v.AccountRef]++
		}
	}

	require.NotEmpty(t, accounts)
	for ref := range accounts {
		assert.Equal(t, 1, extensions[ref], "account %s must have exactly one extension", ref)
	}
}

func TestShouldFoldGroupAttributesIntoExtension(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &GroupEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.GroupEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	byGroup := map[string]map[string]string{}
	for _, e := range emitter.emitted {
		if ext, ok := e.(*entities.GroupExtension); ok {
			values := map[string]string{}
			for _, a := range ext.Attributes {
				values[a.Ref] = a.Value
			}
			byGroup[ext.GroupRef] = values
		}
	}

	require.NotEmpty(t, byGroup)
	assert.Equal(t, "corp", byGroup["grp-1"]["Group Domain"])
	assertNoLegacyCatalogEntities(t, emitter.emitted)
}

func TestShouldFoldPersonAttributesIntoExtension(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &OwnerEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.OwnerEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	var ownExt *entities.PeopleExtension
	for _, e := range emitter.emitted {
		if ext, ok := e.(*entities.PeopleExtension); ok && ext.PersonRef == "own-1" {
			ownExt = ext
		}
	}

	require.NotNil(t, ownExt, "expected a PeopleExtension for own-1")
	found := false
	for _, a := range ownExt.Attributes {
		if a.Ref == "Department" && a.Value == "IT" {
			found = true
		}
	}
	assert.True(t, found, "owner's Department field becomes an extension attribute entry")
	assertNoLegacyCatalogEntities(t, emitter.emitted)
}
