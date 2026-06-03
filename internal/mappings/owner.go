package mappings

import (
	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// MapOwner converts a Hydden owner (identity) row into a catalog Person.
// Owners are global identities that transcend datasources, so they are emitted
// as Persons. Returns nil when the row has no usable identifier.
func MapOwner(row api.Row) *entities.Person {
	personRef := firstNonEmpty(row, "Identity Id", "Id", "Identity Email", "Email")
	if personRef == "" {
		return nil
	}
	person := &entities.Person{
		Metadata:    types.EntityMetadata{Space: spaces.Persons},
		PersonRef:   personRef,
		Name:        getString(row, "Identity Name"),
		DisplayName: getString(row, "Identity Name"),
	}
	if email := firstNonEmpty(row, "Identity Email", "Email"); email != "" {
		person.PrimaryEmail = &types.Email{Address: email}
	}
	return person
}
