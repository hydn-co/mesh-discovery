package mappings

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

func TestShouldMapGroup(t *testing.T) {
	g := MapGroup(api.Row{
		"Group Id": "grp-1", "Group Name": "Admins", "Description": "Admin group",
		"Data Source Name": "ds1-name",
	})
	require.NotNil(t, g)
	assert.Equal(t, spaces.Groups, g.Metadata.Space)
	assert.Equal(t, "grp-1", g.GroupRef)
	assert.Equal(t, "Admins", g.Name)
	assert.Equal(t, "Admin group", g.Description)
}

func TestShouldFallBackGroupRefAndName(t *testing.T) {
	g := MapGroup(api.Row{"Id": "grp-2", "Group Display Name": "Display Only"})
	require.NotNil(t, g)
	assert.Equal(t, "grp-2", g.GroupRef)
	assert.Equal(t, "Display Only", g.Name)
}

func TestShouldReturnNilGroupWhenNoIdentifier(t *testing.T) {
	assert.Nil(t, MapGroup(api.Row{"Description": "orphan"}))
}

func TestShouldReturnGroupDatasourceName(t *testing.T) {
	assert.Equal(t, "ds1-name", GroupDatasourceName(api.Row{"Data Source Name": "ds1-name"}))
}

func TestShouldMapGroupMember(t *testing.T) {
	m := MapGroupMember(api.Row{"Group ID": "grp-1", "Account ID": "acc-1"})
	require.NotNil(t, m)
	assert.Equal(t, spaces.GroupMembers, m.Metadata.Space)
	assert.Equal(t, "grp-1", m.GroupRef)
	assert.Equal(t, "acc-1", m.AccountRef)
}

func TestShouldReturnNilGroupMemberWhenEitherSideMissing(t *testing.T) {
	assert.Nil(t, MapGroupMember(api.Row{"Group ID": "grp-1"}))
	assert.Nil(t, MapGroupMember(api.Row{"Account ID": "acc-1"}))
}
