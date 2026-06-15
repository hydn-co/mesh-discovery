package collectors

import (
	"context"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// groupEntityTypePrefix matches every native group record in the datastore
// (group.windows, group.azure, group.generic, …). The datastore fetch
// prefix-matches entityType, so one streamed call returns all groups across all
// datasources — no per-datasource probing.
const groupEntityTypePrefix = "group"

// membershipEntityType is the datastore edge type for account<->group
// memberships (From = group, To = member account).
const membershipEntityType = "edge.membership"

// collectGroupAttributes streams every native group record from the datastore in
// one prefix-filtered firehose and folds its native attributes into the
// per-group accumulator (keyed by the record id, the group ref). As with
// accounts, there is no group-ref join (no FK); merkle reconciliation owns
// change/delete detection.
func collectGroupAttributes(
	ctx context.Context,
	client discoveryClient,
	attrs *attrAccumulator,
) error {
	return client.FetchEntities(ctx, "", groupEntityTypePrefix, func(e *api.FetchedEntity) error {
		if e.Tombstoned || e.ID == "" {
			return nil
		}
		attrs.add(e.ID, mappings.FlattenFetchedEntity(e))
		return nil
	})
}
