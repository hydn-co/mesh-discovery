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

// AccountEntityCollector emits discovery accounts as catalog Account entities and
// links each to its datasource Application via an ApplicationAccount edge.
type AccountEntityCollector struct {
	*connector.TypedFeatureContext[*options.AccountEntityCollectorOptions, *connector.NoPayload]

	newClient clientFactory
}

func NewAccountEntityCollector(
	ctx *connector.TypedFeatureContext[*options.AccountEntityCollectorOptions, *connector.NoPayload],
) runner.Feature {
	return &AccountEntityCollector{TypedFeatureContext: ctx, newClient: defaultClientFactory}
}

func (c *AccountEntityCollector) Init(_ context.Context) error {
	return connectorutil.Validate(c.GetOptions(), "account collector options")
}

func (c *AccountEntityCollector) Stop(_ context.Context) error { return nil }

func (c *AccountEntityCollector) Start(ctx context.Context) error {
	connectorutil.LogFeature(ctx, c.TypedFeatureContext, slog.LevelInfo, "Starting discovery account collector")

	clientID, clientSecret, err := credentials.ExtractClientCredentials(c.GetCredentials())
	if err != nil {
		return fmt.Errorf("discovery credentials: %w", err)
	}

	client := c.newClient(c.GetOptions().GetBaseURL(), clientID, clientSecret)

	return client.ForEachAccountPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			account := mappings.MapAccount(row)
			if account == nil {
				continue
			}
			if err := c.Emit(ctx, account); err != nil {
				return fmt.Errorf("emit account %s: %w", account.AccountRef, err)
			}
			// Link the account to its datasource application. Accounts carry the
			// datasource id directly, so no name resolution is needed here.
			if edge := mappings.NewApplicationAccount(mappings.DatasourceOf(row).ID, account.AccountRef); edge != nil {
				if err := c.Emit(ctx, edge); err != nil {
					return fmt.Errorf("emit application-account %s: %w", account.AccountRef, err)
				}
			}
		}
		return nil
	})
}
