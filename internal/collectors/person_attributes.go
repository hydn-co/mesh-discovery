package collectors

import (
	"context"

	"github.com/hydn-co/mesh-sdk/pkg/connector"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// personEntityTypePrefix matches every native identity/person record in the
// datastore (identity.auto, identity.*). Owners are emitted as Persons from the
// search grid, but their native attributes live in the datastore under this
// prefix — control never collected these, but they are real, so we stream them
// the same way as accounts and groups.
const personEntityTypePrefix = "identity"

// collectPersonAttributes streams every native identity record from the
// datastore in one prefix-filtered firehose and emits Attribute definitions +
// PersonAttribute value edges. No person-ref join (no FK); merkle reconciliation
// owns change/delete detection.
func collectPersonAttributes(
	ctx context.Context,
	emitter connector.EntityEmitter,
	client discoveryClient,
	seenAttr map[string]struct{},
) error {
	return client.FetchEntities(ctx, "", personEntityTypePrefix, func(e *api.FetchedEntity) error {
		if e.Tombstoned || e.ID == "" {
			return nil
		}
		return emitNamedAttributes(ctx, emitter, mappings.FlattenFetchedEntity(e), seenAttr,
			func(name, value string) any { return mappings.NewPersonAttribute(e.ID, name, value) })
	})
}
