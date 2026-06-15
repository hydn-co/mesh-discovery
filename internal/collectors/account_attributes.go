package collectors

import (
	"context"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// accountEntityTypePrefix matches every native account record in the datastore
// (all subtypes: principal.account.user.*, .serviceaccount.*, .federated.*, …).
// The datastore fetch prefix-matches entityType, so a single streamed call
// returns all accounts across all datasources — no per-account probing, and none
// of the edge/compromise records the unfiltered firehose is dominated by.
const accountEntityTypePrefix = "principal.account"

// collectAccountAttributes streams every native account record from the
// datastore in one prefix-filtered firehose and folds its native attributes into
// the per-account accumulator (keyed by the record id, the account ref). There
// is no account-ref join: the catalog has no FK constraint, so an attribute may
// reference an account that arrives later (or not at all), and merkle
// reconciliation owns change/delete detection.
func collectAccountAttributes(
	ctx context.Context,
	client discoveryClient,
	attrs *attrAccumulator,
) error {
	return client.FetchEntities(ctx, "", accountEntityTypePrefix, func(e *api.FetchedEntity) error {
		if e.Tombstoned || e.ID == "" {
			return nil
		}
		attrs.add(e.ID, mappings.FlattenFetchedEntity(e))
		return nil
	})
}
