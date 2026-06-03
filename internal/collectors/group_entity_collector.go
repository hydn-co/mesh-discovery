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

	// Pass 1: emit groups + application links.
	if err := client.ForEachGroupPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			group := mappings.MapGroup(row)
			if group == nil {
				continue
			}
			if err := c.Emit(ctx, group); err != nil {
				return fmt.Errorf("emit group %s: %w", group.GroupRef, err)
			}
			if appRef := appIDByName[mappings.GroupDatasourceName(row)]; appRef != "" {
				if edge := mappings.NewApplicationGroup(appRef, group.GroupRef); edge != nil {
					if err := c.Emit(ctx, edge); err != nil {
						return fmt.Errorf("emit application-group %s: %w", group.GroupRef, err)
					}
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: emit memberships.
	return client.ForEachGroupMembershipPage(ctx, func(page []api.Row, _, _ int) error {
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
	})
}
