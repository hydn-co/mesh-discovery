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

	// Pass 1: emit accounts + datasource links, indexing accounts by external id
	// (for the attribute join) and bucketing them by datasource (for probing).
	accountRefs := make(map[string]struct{})
	byDatasource := make(map[string][]accountProbe)
	if err := client.ForEachAccountPage(ctx, func(page []api.Row, _, _ int) error {
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
			dsID := mappings.DatasourceOf(row).ID
			if edge := mappings.NewApplicationAccount(dsID, account.AccountRef); edge != nil {
				if err := c.Emit(ctx, edge); err != nil {
					return fmt.Errorf("emit application-account %s: %w", account.AccountRef, err)
				}
			}
			accountRefs[account.AccountRef] = struct{}{}
			if dsID != "" {
				byDatasource[dsID] = append(byDatasource[dsID],
					accountProbe{ref: account.AccountRef, accountType: mappings.AccountTypeRaw(row)})
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: collect each account's full attribute set and emit Attribute
	// definitions + AccountAttribute value edges.
	return collectAccountAttributes(ctx, c, client, accountRefs, byDatasource)
}
