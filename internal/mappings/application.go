package mappings

import (
	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"
)

// MapApplication represents a discovery datasource as a catalog Application:
// ApplicationRef = datasource id, Name = datasource name, Description = platform.
// Returns nil when the datasource has no id.
func MapApplication(ds Datasource) *entities.Application {
	if ds.ID == "" {
		return nil
	}
	return &entities.Application{
		Metadata:       types.EntityMetadata{Space: spaces.Applications},
		ApplicationRef: ds.ID,
		Name:           ds.Name,
		Description:    ds.Platform,
	}
}

// NewApplicationAccount links an account to its datasource application.
func NewApplicationAccount(applicationRef, accountRef string) *entities.ApplicationAccount {
	if applicationRef == "" || accountRef == "" {
		return nil
	}
	return &entities.ApplicationAccount{
		Metadata:       types.EntityMetadata{Space: spaces.ApplicationAccounts},
		ApplicationRef: applicationRef,
		AccountRef:     accountRef,
	}
}

// NewApplicationGroup links a group to its datasource application.
func NewApplicationGroup(applicationRef, groupRef string) *entities.ApplicationGroup {
	if applicationRef == "" || groupRef == "" {
		return nil
	}
	return &entities.ApplicationGroup{
		Metadata:       types.EntityMetadata{Space: spaces.ApplicationGroups},
		ApplicationRef: applicationRef,
		GroupRef:       groupRef,
	}
}

// NewApplicationRole links a role to its datasource application.
func NewApplicationRole(applicationRef, roleRef string) *entities.ApplicationRole {
	if applicationRef == "" || roleRef == "" {
		return nil
	}
	return &entities.ApplicationRole{
		Metadata:       types.EntityMetadata{Space: spaces.ApplicationRoles},
		ApplicationRef: applicationRef,
		RoleRef:        roleRef,
	}
}
