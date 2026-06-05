package collectors

import (
	"context"
	"fmt"

	"github.com/hydn-co/mesh-sdk/pkg/connector"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// collectGroupAttributes streams full group entity records per datasource and
// emits distinct Attribute definitions plus GroupAttribute value edges. Mirrors
// control's group_attributes sync: probe one group per datasource to discover
// its datastore entity type (group.windows, group.azure, …), stream every
// entity of that type via /internal/v1/datastore/fetch, and join records back to
// groups by external id.
func collectGroupAttributes(
	ctx context.Context,
	emitter connector.EntityEmitter,
	client discoveryClient,
	groupRefs map[string]struct{},
	byDatasource map[string][]string,
) error {
	for dsID, groupIDs := range byDatasource {
		if err := ctx.Err(); err != nil {
			return err
		}
		entityType := probeGroupEntityType(ctx, client, dsID, groupIDs)
		if entityType == "" {
			continue
		}
		err := client.FetchEntities(ctx, dsID, entityType, func(e *api.FetchedEntity) error {
			if e.Tombstoned {
				return nil
			}
			if _, ok := groupRefs[e.ID]; !ok {
				return nil
			}
			return emitNamedAttributes(ctx, emitter, mappings.FlattenFetchedEntity(e), nil,
				func(name, value string) any { return mappings.NewGroupAttribute(e.ID, name, value) })
		})
		if err != nil {
			return fmt.Errorf("fetch group entities %s/%s: %w", dsID, entityType, err)
		}
	}
	return nil
}

// probeGroupEntityType reads one group per datasource via the per-record
// endpoint to discover the datastore entity type that datasource emits. Groups
// have no type variants (unlike accounts), so the first successful probe wins.
func probeGroupEntityType(ctx context.Context, client discoveryClient, dsID string, groupIDs []string) string {
	for _, gid := range groupIDs {
		if ctx.Err() != nil {
			return ""
		}
		resp, err := client.GetAccountDetails(ctx, dsID, gid)
		if err != nil {
			continue
		}
		if et, _ := resp["type"].(string); et != "" {
			return et
		}
	}
	return ""
}
