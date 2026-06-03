package mappings

import (
	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// RoleDatasourceName returns the datasource an application-role row belongs to.
// Role rows carry "Data Source Name" (see control's mapping_app_role.go).
func RoleDatasourceName(row api.Row) string {
	return getString(row, "Data Source Name")
}

// MapApplicationRole converts a Hydden application-role (entitlement) row into a
// catalog Role. Returns nil when the row has no usable identifier.
func MapApplicationRole(row api.Row) *entities.Role {
	roleRef := getString(row, "Id")
	if roleRef == "" {
		return nil
	}
	return &entities.Role{
		Metadata:    types.EntityMetadata{Space: spaces.Roles},
		RoleRef:     roleRef,
		Name:        firstNonEmpty(row, "Display Name", "Name"),
		Description: getString(row, "Domain"),
	}
}

// AppRoleMembershipRoleRef returns the role id referenced by a per-account
// app-role membership row ("Role Id", with a lowercase "id" fallback — see
// control's mapping_app_role_membership.go).
func AppRoleMembershipRoleRef(row api.Row) string {
	return firstNonEmpty(row, "Role Id", "id")
}

// MapAccountRole builds an AccountRole linking the given account to the role in
// a per-account app-role membership row. Returns nil when the role id is absent.
func MapAccountRole(accountRef string, membershipRow api.Row) *entities.AccountRole {
	roleRef := AppRoleMembershipRoleRef(membershipRow)
	if accountRef == "" || roleRef == "" {
		return nil
	}
	return &entities.AccountRole{
		Metadata:   types.EntityMetadata{Space: spaces.AccountRoles},
		AccountRef: accountRef,
		RoleRef:    roleRef,
	}
}
