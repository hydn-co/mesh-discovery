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

// GroupEntityCollector collects discovery groups and their memberships. Groups
// are scoped by datasource NAME (group rows carry "Data Source Name", not id);
// memberships are scoped to the groups of the datasource since membership rows
// carry no datasource of their own.
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

	opts := c.GetOptions()
	client := c.newClient(opts.GetBaseURL(), clientID, clientSecret)
	scopeName := opts.GetDataSourceName()

	// Pass 1: emit groups, recording the refs that belong to this datasource so
	// memberships can be scoped to them.
	groupRefs := make(map[string]struct{})
	if err := client.ForEachGroupPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			if scopeName != "" && mappings.GroupDatasourceName(row) != scopeName {
				continue
			}
			group := mappings.MapGroup(row)
			if group == nil {
				continue
			}
			groupRefs[group.GroupRef] = struct{}{}
			if err := c.Emit(ctx, group); err != nil {
				return fmt.Errorf("emit group %s: %w", group.GroupRef, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: emit memberships for groups in scope.
	return client.ForEachGroupMembershipPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			if scopeName != "" {
				if _, ok := groupRefs[mappings.MembershipGroupRef(row)]; !ok {
					continue
				}
			}
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
