package mappings

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

func TestShouldMapOwnerToPerson(t *testing.T) {
	p := MapOwner(api.Row{
		"Identity Id": "own-1", "Identity Name": "Owner One",
		"Identity Email": "owner1@corp.example",
		"Alt Email":      "owner1.alt@corp.example",
		"Mobile Phone":   "555-1234",
	})
	require.NotNil(t, p)
	assert.Equal(t, spaces.Persons, p.Metadata.Space)
	assert.Equal(t, "own-1", p.PersonRef)
	assert.Equal(t, "Owner One", p.Name)
	assert.Equal(t, "Owner One", p.DisplayName)
	require.NotNil(t, p.PrimaryEmail)
	assert.Equal(t, "owner1@corp.example", p.PrimaryEmail.Address)
	require.Len(t, p.AlternateEmails, 1)
	assert.Equal(t, "owner1.alt@corp.example", p.AlternateEmails[0].Address)
	require.NotNil(t, p.PrimaryPhone)
	assert.Equal(t, "555-1234", p.PrimaryPhone.Number)
}

func TestShouldFallBackOwnerRefAndEmail(t *testing.T) {
	p := MapOwner(api.Row{"Id": "own-2", "Email": "x@corp.example"})
	require.NotNil(t, p)
	assert.Equal(t, "own-2", p.PersonRef)
	require.NotNil(t, p.PrimaryEmail)
	assert.Equal(t, "x@corp.example", p.PrimaryEmail.Address)
}

func TestShouldPreferMobilePhoneOverPhone(t *testing.T) {
	p := MapOwner(api.Row{"Identity Id": "own-1", "Phone": "555-0000", "Mobile Phone": "555-1111"})
	require.NotNil(t, p)
	require.NotNil(t, p.PrimaryPhone)
	assert.Equal(t, "555-1111", p.PrimaryPhone.Number)
}

func TestShouldReturnNilOwnerWhenNoIdentifier(t *testing.T) {
	assert.Nil(t, MapOwner(api.Row{"Department": "IT"}))
}
