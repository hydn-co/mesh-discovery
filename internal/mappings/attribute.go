package mappings

import (
	"encoding/json"
	"fmt"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// AccountTypeRaw returns the raw "Account Type" value from an account row, used
// to bucket accounts when probing a datasource's entity-type variants.
func AccountTypeRaw(row api.Row) string {
	return getString(row, "Account Type")
}

// NewAttribute creates a distinct attribute definition. ref is the attribute's
// stable key (the flattened field name).
func NewAttribute(ref string) *entities.Attribute {
	if ref == "" {
		return nil
	}
	return &entities.Attribute{
		Metadata:     types.EntityMetadata{Space: spaces.Attributes},
		AttributeRef: ref,
		Name:         ref,
	}
}

// NewAccountAttribute stores one attribute value for an account.
func NewAccountAttribute(accountRef, attributeRef, value string) *entities.AccountAttribute {
	if accountRef == "" || attributeRef == "" {
		return nil
	}
	return &entities.AccountAttribute{
		Metadata:     types.EntityMetadata{Space: spaces.AccountAttributes},
		AccountRef:   accountRef,
		AttributeRef: attributeRef,
		Value:        value,
	}
}

// NewGroupAttribute stores one attribute value for a group.
func NewGroupAttribute(groupRef, attributeRef, value string) *entities.GroupAttribute {
	if groupRef == "" || attributeRef == "" {
		return nil
	}
	return &entities.GroupAttribute{
		Metadata:     types.EntityMetadata{Space: spaces.GroupAttributes},
		GroupRef:     groupRef,
		AttributeRef: attributeRef,
		Value:        value,
	}
}

// NewPersonAttribute stores one attribute value for a person.
func NewPersonAttribute(personRef, attributeRef, value string) *entities.PersonAttribute {
	if personRef == "" || attributeRef == "" {
		return nil
	}
	return &entities.PersonAttribute{
		Metadata:     types.EntityMetadata{Space: spaces.PersonAttributes},
		PersonRef:    personRef,
		AttributeRef: attributeRef,
		Value:        value,
	}
}

// accountAttributeSkip lists the account-grid columns that are NOT emitted as
// attributes: those folded onto the typed Account entity, used to identify the
// datasource (the Application), or consumed as risk factors / classifications.
var accountAttributeSkip = func() map[string]struct{} {
	skip := map[string]struct{}{
		// Folded onto the typed Account.
		"Id": {}, "Email": {}, "Account Name": {}, "Display Name": {},
		"Account Type": {}, "UPN": {}, "Status": {}, "Created": {},
		"Last Logon": {}, "Disabled Time": {},
		// Identify the datasource (modeled as the Application, not an attribute).
		"Data Source Id": {}, "Data Source Name": {}, "Data Source Platform": {},
	}
	for k := range classificationGSKeys {
		skip[k] = struct{}{}
	}
	for k := range riskFactorGSKeys() {
		skip[k] = struct{}{}
	}
	return skip
}()

// groupAttributeSkip lists group-grid columns folded onto the typed Group or
// used to resolve the datasource Application link.
var groupAttributeSkip = map[string]struct{}{
	"Group Id": {}, "Id": {}, "Group Name": {}, "Description": {},
	"Data Source Name": {}, "Data Source Platform": {},
}

// personAttributeSkip lists owner-grid columns folded onto the typed Person.
var personAttributeSkip = map[string]struct{}{
	"Identity Id": {}, "Id": {}, "Identity Name": {}, "Identity Email": {},
	"Email": {}, "Alt Email": {}, "Phone": {}, "Mobile Phone": {},
}

// AccountGSAttributes returns the account-grid fields kept as attributes — the
// "remaining fields" once typed columns, datasource, risk factors and
// classifications are removed.
func AccountGSAttributes(row api.Row) map[string]string {
	return flattenRowExcept(row, accountAttributeSkip)
}

// GroupGSAttributes returns the group-grid fields kept as attributes.
func GroupGSAttributes(row api.Row) map[string]string {
	return flattenRowExcept(row, groupAttributeSkip)
}

// PersonGSAttributes returns the owner-grid fields kept as attributes.
func PersonGSAttributes(row api.Row) map[string]string {
	return flattenRowExcept(row, personAttributeSkip)
}

// flattenRowExcept drops the skip columns from a discovery row, then flattens
// the remainder to dot-notation field-name -> string-value pairs (nested values
// like an owner's "Roles" list surface as "Roles.0", "Roles.1", ...). Empty
// values are omitted.
func flattenRowExcept(row api.Row, skip map[string]struct{}) map[string]string {
	filtered := make(map[string]any, len(row))
	for k, v := range row {
		if _, ok := skip[k]; ok {
			continue
		}
		filtered[k] = v
	}
	out := make(map[string]string)
	flattenJSON(filtered, "", out)
	for k, v := range out {
		if v == "" {
			delete(out, k)
		}
	}
	return out
}

// FlattenFetchedEntity flattens a datastore fetch record into dot-notation
// field-name -> string-value pairs. The whole envelope is flattened (matching
// control's account_attributes sync), so payload fields surface as "entity.<k>".
func FlattenFetchedEntity(e *api.FetchedEntity) map[string]string {
	b, err := json.Marshal(e)
	if err != nil {
		return map[string]string{}
	}
	var envelope map[string]any
	if err := json.Unmarshal(b, &envelope); err != nil {
		return map[string]string{}
	}
	out := make(map[string]string)
	flattenJSON(envelope, "", out)
	return out
}

// flattenJSON recursively flattens nested maps/slices to dot-notation keys with
// stringified scalar values (port of control's common.FlattenJSON).
func flattenJSON(data any, prefix string, out map[string]string) {
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "." + key
			}
			flattenJSON(value, newKey, out)
		}
	case []any:
		for i, value := range v {
			flattenJSON(value, fmt.Sprintf("%s.%d", prefix, i), out)
		}
	case nil:
		out[prefix] = ""
	default:
		out[prefix] = fmt.Sprintf("%v", v)
	}
}
