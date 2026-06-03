package mappings

import (
	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// GroupDatasourceName returns the datasource a group row belongs to. Group rows
// carry "Data Source Name" (not "Data Source Id"), so groups/roles are scoped
// by datasource NAME rather than id (see control's mapping_group.go).
func GroupDatasourceName(row api.Row) string {
	return getString(row, "Data Source Name")
}

// MapGroup converts a Hydden group row into a catalog Group. Returns nil when
// the row has no usable identifier.
func MapGroup(row api.Row) *entities.Group {
	groupRef := firstNonEmpty(row, "Group Id", "Id", "Group Name")
	if groupRef == "" {
		return nil
	}
	return &entities.Group{
		Metadata:    types.EntityMetadata{Space: spaces.Groups},
		GroupRef:    groupRef,
		Name:        firstNonEmpty(row, "Group Name", "Group Display Name"),
		Description: getString(row, "Description"),
	}
}

// MapGroupMember converts a Hydden group-membership row into a GroupMember.
// Membership rows use "Group ID"/"Account ID" (note the upper-case "ID", which
// differs from the "Group Id"/"Id" keys on the group and account feeds, but the
// id VALUES match). Returns nil when either side is missing.
func MapGroupMember(row api.Row) *entities.GroupMember {
	groupRef := getString(row, "Group ID")
	accountRef := getString(row, "Account ID")
	if groupRef == "" || accountRef == "" {
		return nil
	}
	return &entities.GroupMember{
		Metadata:   types.EntityMetadata{Space: spaces.GroupMembers},
		GroupRef:   groupRef,
		AccountRef: accountRef,
	}
}

// MembershipGroupRef returns the group id referenced by a membership row, used
// to scope memberships to the groups of a single datasource.
func MembershipGroupRef(row api.Row) string {
	return getString(row, "Group ID")
}
