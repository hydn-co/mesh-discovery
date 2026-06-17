package collectors

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"
	"github.com/hydn-co/mesh-sdk/pkg/connector"
	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
	"github.com/hydn-co/mesh-sdk/pkg/runner"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/credentials"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
	"github.com/hydn-co/mesh-discovery/internal/options"
)

// GroupEntityCollector emits discovery groups and their memberships, and links
// each group to its datasource Application via an ApplicationGroup edge. Group
// rows carry the datasource NAME, so the application id is resolved through the
// name->id index built from the account feed.
type GroupEntityCollector struct {
	*connector.TypedFeatureContext[*options.GroupEntityCollectorOptions, *connector.NoPayload]

	newClient clientFactory
}

func NewGroupEntityCollector(
	ctx *connector.TypedFeatureContext[*options.GroupEntityCollectorOptions, *connector.NoPayload],
) runner.Feature {
	return &GroupEntityCollector{TypedFeatureContext: ctx, newClient: defaultClientFactory}
}

func (c *GroupEntityCollector) Init(_ context.Context) error {
	return connectorutil.Validate(c.GetOptions(), "group collector options")
}

func (c *GroupEntityCollector) Stop(_ context.Context) error { return nil }

func (c *GroupEntityCollector) Start(ctx context.Context) error {
	connectorutil.LogFeature(ctx, c.TypedFeatureContext, slog.LevelInfo, "Starting discovery group collector")

	clientID, clientSecret, err := credentials.ExtractClientCredentials(c.GetCredentials())
	if err != nil {
		return fmt.Errorf("discovery credentials: %w", err)
	}

	client := c.newClient(c.GetOptions().GetBaseURL(), clientID, clientSecret)

	appIDByName, err := datasourceIDByName(ctx, client)
	if err != nil {
		return fmt.Errorf("build datasource index: %w", err)
	}

	// Pass 1: emit groups and application links, and accumulate each group's
	// grid-enriched attributes. The consolidated GroupExtension is emitted after
	// Pass 3 so it carries attributes from both the grid and the datastore
	// (hydn-co/control#1436).
	attrs := newAttrAccumulator()
	if err := client.ForEachGroupPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			group := mappings.MapGroup(row)
			if group == nil {
				continue
			}
			if err := c.Emit(ctx, group); err != nil {
				return fmt.Errorf("emit group %s: %w", group.GroupRef, err)
			}
			if dsID := appIDByName[mappings.GroupDatasourceName(row)]; dsID != "" {
				if edge := mappings.NewApplicationGroup(dsID, group.GroupRef); edge != nil {
					if err := c.Emit(ctx, edge); err != nil {
						return fmt.Errorf("emit application-group %s: %w", group.GroupRef, err)
					}
				}
			}
			attrs.add(group.GroupRef, mappings.GroupGSAttributes(row))
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: emit account<->group memberships from the edge.membership stream —
	// one prefix-filtered firehose, the same pattern as edge.role. edge.From is
	// the group external id, edge.To the member account external id.
	if err := client.FetchEntities(ctx, "", membershipEntityType, func(edge *api.FetchedEntity) error {
		if edge.Tombstoned {
			return nil
		}
		member := mappings.NewGroupMember(edge.From, edge.To)
		if member == nil {
			return nil
		}
		if err := c.Emit(ctx, member); err != nil {
			return fmt.Errorf("emit group member %s/%s: %w", member.GroupRef, member.AccountRef, err)
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 3: stream every native group record from the datastore in a single
	// prefix-filtered firehose and fold its native attributes into the per-group
	// accumulator.
	if err := collectGroupAttributes(ctx, client, attrs); err != nil {
		return err
	}

	// Emit one consolidated GroupExtension per group; mesh-core shards oversized
	// payloads transparently (hydn-co/mesh-core#271).
	for _, ref := range attrs.refs() {
		ext := &entities.GroupExtension{
			Metadata:   types.EntityMetadata{Space: spaces.GroupExtensions},
			GroupRef:   ref,
			Attributes: attrs.attributesFor(ref),
		}
		if err := c.Emit(ctx, ext); err != nil {
			return fmt.Errorf("emit group extension %s: %w", ref, err)
		}
	}

	return nil
}
