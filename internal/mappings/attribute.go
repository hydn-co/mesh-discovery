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
