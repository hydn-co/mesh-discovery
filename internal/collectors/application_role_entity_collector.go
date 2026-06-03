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

// ApplicationRoleEntityCollector emits discovery application roles (entitlements)
// as catalog Role entities, links each to its datasource Application via an
// ApplicationRole edge, and emits account<->role memberships as AccountRole
// edges. Role rows carry the datasource NAME (resolved to an application id via
// the account-feed index); memberships stream as edge.role records.
type ApplicationRoleEntityCollector struct {
	*connector.TypedFeatureContext[*options.ApplicationRoleEntityCollectorOptions, *connector.NoPayload]

	newClient clientFactory
}

func NewApplicationRoleEntityCollector(
	ctx *connector.TypedFeatureContext[*options.ApplicationRoleEntityCollectorOptions, *connector.NoPayload],
) runner.Feature {
	return &ApplicationRoleEntityCollector{TypedFeatureContext: ctx, newClient: defaultClientFactory}
}

func (c *ApplicationRoleEntityCollector) Init(_ context.Context) error {
	return connectorutil.Validate(c.GetOptions(), "application role collector options")
}

func (c *ApplicationRoleEntityCollector) Stop(_ context.Context) error { return nil }

func (c *ApplicationRoleEntityCollector) Start(ctx context.Context) error {
	connectorutil.LogFeature(
		ctx,
		c.TypedFeatureContext,
		slog.LevelInfo,
		"Starting discovery application role collector",
	)

	clientID, clientSecret, err := credentials.ExtractClientCredentials(c.GetCredentials())
	if err != nil {
		return fmt.Errorf("discovery credentials: %w", err)
	}

	client := c.newClient(c.GetOptions().GetBaseURL(), clientID, clientSecret)

	appIDByName, err := datasourceIDByName(ctx, client)
	if err != nil {
		return fmt.Errorf("build datasource index: %w", err)
	}

	// Pass 1: emit roles + application links.
	if err := client.ForEachApplicationRolePage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			role := mappings.MapApplicationRole(row)
			if role == nil {
				continue
			}
			if err := c.Emit(ctx, role); err != nil {
				return fmt.Errorf("emit role %s: %w", role.RoleRef, err)
			}
			if appRef := appIDByName[mappings.RoleDatasourceName(row)]; appRef != "" {
				if edge := mappings.NewApplicationRole(appRef, role.RoleRef); edge != nil {
					if err := c.Emit(ctx, edge); err != nil {
						return fmt.Errorf("emit application-role %s: %w", role.RoleRef, err)
					}
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: emit account<->role memberships from the edge.role stream
	// (edge.From = role external id, edge.To = account external id).
	return client.FetchEntities(ctx, "", "edge.role", func(edge *api.FetchedEntity) error {
		if edge.Tombstoned {
			return nil
		}
		accountRole := mappings.NewAccountRole(edge.To, edge.From)
		if accountRole == nil {
			return nil
		}
		if err := c.Emit(ctx, accountRole); err != nil {
			return fmt.Errorf("emit account role %s/%s: %w", accountRole.AccountRef, accountRole.RoleRef, err)
		}
		return nil
	})
}
