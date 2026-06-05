package collectors

import (
	"context"
	"fmt"

	"github.com/hydn-co/mesh-sdk/pkg/connector"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// accountTypeProbeCap bounds how many distinct AccountType values are probed per
// datasource to discover its entity-type variants (mirrors control).
const accountTypeProbeCap = 16

// accountProbe pairs an account's reference with its raw AccountType, used to
// probe a datasource's entity-type variants.
type accountProbe struct {
	ref         string
	accountType string
}

// collectAccountAttributes streams full account entity records per datasource
// and emits distinct Attribute definitions plus AccountAttribute value edges.
// Mirrors control's account_attributes sync: probe one account per distinct
// AccountType to discover the datasource's entity-type variants, stream each
// type via /internal/v1/datastore/fetch, and join records back to accounts by
// external id. Distinct attribute names are emitted once as Attribute entities.
func collectAccountAttributes(
	ctx context.Context,
	emitter connector.EntityEmitter,
	client discoveryClient,
	accountRefs map[string]struct{},
	byDatasource map[string][]accountProbe,
	seenAttr map[string]struct{},
) error {
	for dsID, probes := range byDatasource {
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, entityType := range discoverEntityTypes(ctx, client, dsID, probes) {
			if err := ctx.Err(); err != nil {
				return err
			}
			err := client.FetchEntities(ctx, dsID, entityType, func(e *api.FetchedEntity) error {
				if e.Tombstoned {
					return nil
				}
				if _, ok := accountRefs[e.ID]; !ok {
					return nil
				}
				return emitAttributes(ctx, emitter, e, seenAttr)
			})
			if err != nil {
				return fmt.Errorf("fetch entities %s/%s: %w", dsID, entityType, err)
			}
		}
	}
	return nil
}

// emitAttributes flattens one fetched record and emits its attribute definitions
// (once each, via the shared dedupe set) and the account's attribute values.
func emitAttributes(
	ctx context.Context,
	emitter connector.EntityEmitter,
	e *api.FetchedEntity,
	seenAttr map[string]struct{},
) error {
	return emitNamedAttributes(ctx, emitter, mappings.FlattenFetchedEntity(e), seenAttr,
		func(name, value string) any { return mappings.NewAccountAttribute(e.ID, name, value) })
}

// discoverEntityTypes probes one account per distinct AccountType (capped) and
// returns the set of datastore entity-type strings the datasource emits.
func discoverEntityTypes(ctx context.Context, client discoveryClient, dsID string, probes []accountProbe) []string {
	buckets := make(map[string]string, accountTypeProbeCap)
	for _, p := range probes {
		if _, ok := buckets[p.accountType]; !ok {
			buckets[p.accountType] = p.ref
			if len(buckets) >= accountTypeProbeCap {
				break
			}
		}
	}

	typeSet := make(map[string]struct{}, len(buckets))
	for _, ref := range buckets {
		if ctx.Err() != nil {
			break
		}
		resp, err := client.GetAccountDetails(ctx, dsID, ref)
		if err != nil {
			continue
		}
		if et, _ := resp["type"].(string); et != "" {
			typeSet[et] = struct{}{}
		}
	}

	out := make([]string, 0, len(typeSet))
	for t := range typeSet {
		out = append(out, t)
	}
	return out
}
