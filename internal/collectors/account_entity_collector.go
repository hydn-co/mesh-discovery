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

// AccountEntityCollector collects discovery accounts, optionally scoped to a
// single datasource, and emits them as catalog Account entities.
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

	opts := c.GetOptions()
	client := c.newClient(opts.GetBaseURL(), clientID, clientSecret)
	scope := opts.GetDataSourceID()

	return client.ForEachAccountPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			if scope != "" && mappings.DatasourceOf(row).ID != scope {
				continue
			}
			account := mappings.MapAccount(row)
			if account == nil {
				continue
			}
			if err := c.Emit(ctx, account); err != nil {
				return fmt.Errorf("emit account %s: %w", account.AccountRef, err)
			}
		}
		return nil
	})
}
