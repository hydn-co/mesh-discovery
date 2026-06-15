package collectors

import (
	"context"

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
// datastore in one prefix-filtered firehose and folds its native attributes into
// the per-person accumulator (keyed by the record id, the person ref). No
// person-ref join (no FK); merkle reconciliation owns change/delete detection.
func collectPersonAttributes(
	ctx context.Context,
	client discoveryClient,
	attrs *attrAccumulator,
) error {
	return client.FetchEntities(ctx, "", personEntityTypePrefix, func(e *api.FetchedEntity) error {
		if e.Tombstoned || e.ID == "" {
			return nil
		}
		attrs.add(e.ID, mappings.FlattenFetchedEntity(e))
		return nil
	})
}
