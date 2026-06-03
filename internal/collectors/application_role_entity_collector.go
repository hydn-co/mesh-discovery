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

// ApplicationRoleEntityCollector collects discovery application roles
// (entitlements) and per-account role memberships, emitting Role and
// AccountRole entities. Roles are scoped by datasource NAME; memberships are
// fetched per account (scoped by datasource id) and filtered to roles in scope.
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

	opts := c.GetOptions()
	client := c.newClient(opts.GetBaseURL(), clientID, clientSecret)
	scopeName := opts.GetDataSourceName()
	scopeID := opts.GetDataSourceID()

	// Pass 1: emit roles (entitlements) for this datasource.
	if err := client.ForEachApplicationRolePage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			if scopeName != "" && mappings.RoleDatasourceName(row) != scopeName {
				continue
			}
			role := mappings.MapApplicationRole(row)
			if role == nil {
				continue
			}
			if err := c.Emit(ctx, role); err != nil {
				return fmt.Errorf("emit role %s: %w", role.RoleRef, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: emit per-account role memberships. App-role memberships are only
	// available per account, so iterate accounts in scope and fetch each.
	return client.ForEachAccountPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			if scopeID != "" && mappings.DatasourceOf(row).ID != scopeID {
				continue
			}
			accountRef := mappings.AccountRef(row)
			if accountRef == "" {
				continue
			}
			memberships, err := client.GetAccountAppRoles(ctx, accountRef)
			if err != nil {
				return fmt.Errorf("fetch app roles for account %s: %w", accountRef, err)
			}
			for _, mrow := range memberships {
				if scopeName != "" && mappings.RoleDatasourceName(mrow) != scopeName {
					continue
				}
				accountRole := mappings.MapAccountRole(accountRef, mrow)
				if accountRole == nil {
					continue
				}
				if err := c.Emit(ctx, accountRole); err != nil {
					return fmt.Errorf("emit account role %s/%s: %w", accountRole.AccountRef, accountRole.RoleRef, err)
				}
			}
		}
		return nil
	})
}
