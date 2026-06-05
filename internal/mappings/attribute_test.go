package mappings

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

func TestShouldKeepOnlyRemainingAccountFieldsAsAttributes(t *testing.T) {
	row := api.Row{
		// Typed columns (skipped).
		"Id": "acc-1", "Account Name": "Alice", "Status": "Enabled", "UPN": "a@x",
		// Datasource (skipped — modeled as the Application).
		"Data Source Id": "ds1", "Data Source Name": "ds1-name", "Data Source Platform": "AD",
		// Risk factor (skipped — modeled as a RiskFactor).
		"Accounts with MFA Not Enabled": "5",
		// Classification (skipped — modeled as a Classification).
		"Classifications": "Admin", "Is Privileged": "true",
		// Remaining -> attributes.
		"Department": "IT", "Total Threat": "42", "Custom 1": "x",
	}

	attrs := AccountGSAttributes(row)
	assert.Equal(t, map[string]string{
		"Department":   "IT",
		"Total Threat": "42",
		"Custom 1":     "x",
	}, attrs)
}

func TestShouldDropEmptyAttributeValues(t *testing.T) {
	attrs := AccountGSAttributes(api.Row{"Id": "acc-1", "Department": "", "Title": "Eng"})
	_, hasEmpty := attrs["Department"]
	assert.False(t, hasEmpty)
	assert.Equal(t, "Eng", attrs["Title"])
}

func TestShouldKeepRemainingGroupFieldsAsAttributes(t *testing.T) {
	attrs := GroupGSAttributes(api.Row{
		"Group Id": "grp-1", "Group Name": "Admins", "Description": "d",
		"Data Source Name": "ds1-name",
		"Group Domain":     "corp", "Is High Privilege": "true",
	})
	assert.Equal(t, map[string]string{
		"Group Domain":      "corp",
		"Is High Privilege": "true",
	}, attrs)
}

func TestShouldKeepRemainingPersonFieldsAsAttributes(t *testing.T) {
	attrs := PersonGSAttributes(api.Row{
		"Identity Id": "own-1", "Identity Name": "Owner One",
		"Identity Email": "o@x", "Email": "o@x", "Phone": "555",
		"Department": "IT", "Owner Type": "Employee",
	})
	assert.Equal(t, map[string]string{
		"Department": "IT",
		"Owner Type": "Employee",
	}, attrs)
}

func TestShouldFlattenNestedRowValuesIntoAttributes(t *testing.T) {
	attrs := PersonGSAttributes(api.Row{
		"Identity Id": "own-1",
		"Roles":       []any{"admin", "auditor"},
	})
	assert.Equal(t, "admin", attrs["Roles.0"])
	assert.Equal(t, "auditor", attrs["Roles.1"])
}

func TestShouldBuildAttributeDefinition(t *testing.T) {
	assert.Nil(t, NewAttribute(""))
	def := NewAttribute("Department")
	require.NotNil(t, def)
	assert.Equal(t, "Department", def.AttributeRef)
	assert.Equal(t, "Department", def.Name)
	assert.Equal(t, spaces.Attributes, def.Metadata.Space)
}

func TestShouldBuildAttributeValueEdges(t *testing.T) {
	assert.Nil(t, NewAccountAttribute("", "k", "v"))
	assert.Nil(t, NewAccountAttribute("acc-1", "", "v"))
	assert.Nil(t, NewGroupAttribute("grp-1", "", "v"))
	assert.Nil(t, NewPersonAttribute("", "k", "v"))

	acc := NewAccountAttribute("acc-1", "Department", "IT")
	require.NotNil(t, acc)
	assert.Equal(t, spaces.AccountAttributes, acc.Metadata.Space)
	assert.Equal(t, "acc-1", acc.AccountRef)
	assert.Equal(t, "Department", acc.AttributeRef)
	assert.Equal(t, "IT", acc.Value)

	grp := NewGroupAttribute("grp-1", "Group Domain", "corp")
	require.NotNil(t, grp)
	assert.Equal(t, spaces.GroupAttributes, grp.Metadata.Space)
	assert.Equal(t, "grp-1", grp.GroupRef)

	per := NewPersonAttribute("own-1", "Department", "IT")
	require.NotNil(t, per)
	assert.Equal(t, spaces.PersonAttributes, per.Metadata.Space)
	assert.Equal(t, "own-1", per.PersonRef)
}

func TestShouldFlattenFetchedEntity(t *testing.T) {
	flat := FlattenFetchedEntity(&api.FetchedEntity{
		ID:     "acc-1",
		Type:   "principal.account.user",
		Entity: map[string]any{"department": "IT", "nested": map[string]any{"k": "v"}},
	})
	assert.Equal(t, "acc-1", flat["id"])
	assert.Equal(t, "IT", flat["entity.department"])
	assert.Equal(t, "v", flat["entity.nested.k"])
}
