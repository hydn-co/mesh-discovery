package collectors

import (
	"context"
	"fmt"
	"log/slog"

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

	// Pass 1: emit groups, application links, and grid-derived attributes. Index
	// groups by external id (for the fetch join) and bucket them by datasource id
	// (for entity-type probing). seenAttr dedupes the Attribute definitions this
	// run emits into the additive "attributes" dictionary (shared, never pruned).
	groupRefs := make(map[string]struct{})
	byDatasource := make(map[string][]string)
	seenAttr := make(map[string]struct{})
	if err := client.ForEachGroupPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			group := mappings.MapGroup(row)
			if group == nil {
				continue
			}
			if err := c.Emit(ctx, group); err != nil {
				return fmt.Errorf("emit group %s: %w", group.GroupRef, err)
			}
			dsID := appIDByName[mappings.GroupDatasourceName(row)]
			if dsID != "" {
				if edge := mappings.NewApplicationGroup(dsID, group.GroupRef); edge != nil {
					if err := c.Emit(ctx, edge); err != nil {
						return fmt.Errorf("emit application-group %s: %w", group.GroupRef, err)
					}
				}
			}
			if err := emitNamedAttributes(
				ctx,
				c,
				mappings.GroupGSAttributes(row),
				seenAttr,
				func(name, value string) any { return mappings.NewGroupAttribute(group.GroupRef, name, value) },
			); err != nil {
				return err
			}
			groupRefs[group.GroupRef] = struct{}{}
			if dsID != "" {
				byDatasource[dsID] = append(byDatasource[dsID], group.GroupRef)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: emit memberships.
	if err := client.ForEachGroupMembershipPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			member := mappings.MapGroupMember(row)
			if member == nil {
				continue
			}
			if err := c.Emit(ctx, member); err != nil {
				return fmt.Errorf("emit group member %s/%s: %w", member.GroupRef, member.AccountRef, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 3: collect each group's full (native) attribute set from the datastore
	// and emit Attribute definitions + GroupAttribute value edges.
	return collectGroupAttributes(ctx, c, client, groupRefs, byDatasource, seenAttr)
}
