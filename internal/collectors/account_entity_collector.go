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

	// Pass 1: emit accounts, datasource links, and the account-scoped derived
	// graph (risk factors, classifications, and grid-derived attributes). Index
	// accounts by external id (for the fetch attribute join) and bucket them by
	// datasource (for entity-type probing). seen* dedupe the definition entities
	// (Attribute/RiskFactor/Classification) whose refs recur across accounts; the
	// attribute set is shared with Pass 2 so grid and fetched names don't clash.
	accountRefs := make(map[string]struct{})
	byDatasource := make(map[string][]accountProbe)
	seen := accountSeen{
		attr:  make(map[string]struct{}),
		risk:  make(map[string]struct{}),
		class: make(map[string]struct{}),
	}
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
			if err := c.emitAccountDerived(ctx, row, account.AccountRef, seen); err != nil {
				return err
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

	// Pass 2: collect each account's full (native) attribute set from the
	// datastore and emit Attribute definitions + AccountAttribute value edges,
	// sharing the Pass 1 attribute-definition dedupe set.
	return collectAccountAttributes(ctx, c, client, accountRefs, byDatasource, seen.attr)
}

// accountSeen holds the dedupe sets for the definition entities the account
// collector emits across many account rows.
type accountSeen struct {
	attr  map[string]struct{}
	risk  map[string]struct{}
	class map[string]struct{}
}

// emitAccountDerived emits the account-scoped risk factors, classifications, and
// grid-derived attributes for one account row. Definition entities are emitted
// once per distinct ref via the shared dedupe sets; the per-account edges always
// emit.
func (c *AccountEntityCollector) emitAccountDerived(
	ctx context.Context,
	row api.Row,
	accountRef string,
	seen accountSeen,
) error {
	riskDefs, riskEdges := mappings.AccountRiskFactors(row)
	for _, def := range riskDefs {
		if _, ok := seen.risk[def.RiskFactorRef]; ok {
			continue
		}
		seen.risk[def.RiskFactorRef] = struct{}{}
		if err := c.Emit(ctx, def); err != nil {
			return fmt.Errorf("emit risk factor %s: %w", def.RiskFactorRef, err)
		}
	}
	for _, edge := range riskEdges {
		if err := c.Emit(ctx, edge); err != nil {
			return fmt.Errorf("emit account risk factor %s/%s: %w", accountRef, edge.RiskFactorRef, err)
		}
	}

	classDefs, classEdges := mappings.AccountClassifications(row)
	for _, def := range classDefs {
		if _, ok := seen.class[def.ClassificationRef]; ok {
			continue
		}
		seen.class[def.ClassificationRef] = struct{}{}
		if err := c.Emit(ctx, def); err != nil {
			return fmt.Errorf("emit classification %s: %w", def.ClassificationRef, err)
		}
	}
	for _, edge := range classEdges {
		if err := c.Emit(ctx, edge); err != nil {
			return fmt.Errorf("emit account classification %s/%s: %w", accountRef, edge.ClassificationRef, err)
		}
	}

	return emitNamedAttributes(ctx, c, mappings.AccountGSAttributes(row), seen.attr,
		func(name, value string) any { return mappings.NewAccountAttribute(accountRef, name, value) })
}
