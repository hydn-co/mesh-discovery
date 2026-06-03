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

// NewAccountRole builds an AccountRole linking an account to a role. Refs come
// from an edge.role record streamed via FetchEntities (edge.From = role id,
// edge.To = account id), matching control's app-role-membership sync. Returns
// nil when either ref is empty.
func NewAccountRole(accountRef, roleRef string) *entities.AccountRole {
	if accountRef == "" || roleRef == "" {
		return nil
	}
	return &entities.AccountRole{
		Metadata:   types.EntityMetadata{Space: spaces.AccountRoles},
		AccountRef: accountRef,
		RoleRef:    roleRef,
	}
}
