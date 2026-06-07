package collectors

import (
	"context"

	"github.com/hydn-co/mesh-sdk/pkg/connector"

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
// datastore in one prefix-filtered firehose and emits Attribute definitions +
// AccountAttribute value edges. There is no account-ref join: the catalog has no
// FK constraint, so an attribute may reference an account that arrives later (or
// not at all), and merkle reconciliation owns change/delete detection. The only
// in-memory state is the shared Attribute-definition dedupe set.
func collectAccountAttributes(
	ctx context.Context,
	emitter connector.EntityEmitter,
	client discoveryClient,
	seenAttr map[string]struct{},
) error {
	return client.FetchEntities(ctx, "", accountEntityTypePrefix, func(e *api.FetchedEntity) error {
		if e.Tombstoned || e.ID == "" {
			return nil
		}
		return emitNamedAttributes(ctx, emitter, mappings.FlattenFetchedEntity(e), seenAttr,
			func(name, value string) any { return mappings.NewAccountAttribute(e.ID, name, value) })
	})
}
