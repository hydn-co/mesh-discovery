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

func TestShouldEmitAccountRiskFactorsAtFullConfidence(t *testing.T) {
	emitted := runAccountCollector(t)

	var (
		defs  []*entities.RiskFactor
		edges []*entities.AccountRiskFactor
	)
	for _, e := range emitted {
		switch v := e.(type) {
		case *entities.RiskFactor:
			defs = append(defs, v)
		case *entities.AccountRiskFactor:
			edges = append(edges, v)
		}
	}

	require.Len(t, defs, 1, "acc-1 has exactly one risk indicator in the fixture")
	assert.Equal(t, "mfa-not-enabled", defs[0].RiskFactorRef)
	assert.Equal(t, "Password & Security", defs[0].Category)
	assert.Equal(t, 5.0, defs[0].Weight)

	require.Len(t, edges, 1)
	assert.Equal(t, "acc-1", edges[0].AccountRef)
	assert.Equal(t, "mfa-not-enabled", edges[0].RiskFactorRef)
	assert.Equal(t, 1.0, edges[0].Confidence)
}

func TestShouldEmitAccountClassificationsAtFullConfidence(t *testing.T) {
	emitted := runAccountCollector(t)

	var edges []*entities.AccountClassification
	defs := map[string]struct{}{}
	for _, e := range emitted {
		switch v := e.(type) {
		case *entities.Classification:
			defs[v.ClassificationRef] = struct{}{}
		case *entities.AccountClassification:
			edges = append(edges, v)
		}
	}

	_, ok := defs["admin"]
	assert.True(t, ok, "Classifications=\"Admin\" yields the admin classification")
	require.Len(t, edges, 1)
	assert.Equal(t, "acc-1", edges[0].AccountRef)
	assert.Equal(t, "admin", edges[0].ClassificationRef)
	assert.Equal(t, 1.0, edges[0].Confidence)
}

func TestShouldEmitAccountAttributeDefinitionsAndValues(t *testing.T) {
	emitted := runAccountCollector(t)

	var defs, vals int
	for _, e := range emitted {
		switch e.(type) {
		case *entities.Attribute:
			defs++
		case *entities.AccountAttribute:
			vals++
		}
	}
	assert.Positive(t, defs, "the account collector owns the attribute dictionary")
	assert.Positive(t, vals)
}

func TestShouldEmitGroupAttributeDefinitionsAndValues(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &GroupEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.GroupEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	defs := map[string]struct{}{}
	var values []*entities.GroupAttribute
	for _, e := range emitter.emitted {
		switch v := e.(type) {
		case *entities.GroupAttribute:
			values = append(values, v)
		case *entities.Attribute:
			defs[v.AttributeRef] = struct{}{}
		}
	}

	require.NotEmpty(t, values)
	byGroup := map[string]map[string]string{}
	for _, v := range values {
		if byGroup[v.GroupRef] == nil {
			byGroup[v.GroupRef] = map[string]string{}
		}
		byGroup[v.GroupRef][v.AttributeRef] = v.Value
	}
	assert.Equal(t, "corp", byGroup["grp-1"]["Group Domain"])
	// The group collector emits the named definition for each attribute it sets.
	_, ok := defs["Group Domain"]
	assert.True(t, ok, "group collector emits the Attribute definition for Group Domain")
}

func TestShouldEmitPersonAttributeDefinitionsAndValues(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &OwnerEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.OwnerEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	defs := map[string]struct{}{}
	var values []*entities.PersonAttribute
	for _, e := range emitter.emitted {
		switch v := e.(type) {
		case *entities.PersonAttribute:
			values = append(values, v)
		case *entities.Attribute:
			defs[v.AttributeRef] = struct{}{}
		}
	}

	require.NotEmpty(t, values)
	assert.Equal(t, "own-1", values[0].PersonRef)
	found := false
	for _, v := range values {
		if v.AttributeRef == "Department" && v.Value == "IT" {
			found = true
		}
	}
	assert.True(t, found, "owner's Department field becomes a PersonAttribute")
	_, ok := defs["Department"]
	assert.True(t, ok, "owner collector emits the Attribute definition for Department")
}
