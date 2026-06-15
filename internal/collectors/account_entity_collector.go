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
	// graph from the search grid — entities + grid-enriched attributes (computed
	// fields not present in the datastore), classifications, and risk factors. The
	// consolidated AccountExtension is emitted after Pass 2 so it carries
	// attributes from both the grid and the datastore (hydn-co/control#1436).
	attrs := newAttrAccumulator()
	classByRef := make(map[string][]entities.ClassificationEntry)
	riskByRef := make(map[string][]entities.RiskFactorEntry)
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
			attrs.add(account.AccountRef, mappings.AccountGSAttributes(row))
			if class := mappings.AccountClassificationEntries(row); class != nil {
				classByRef[account.AccountRef] = class
			}
			if risk := mappings.AccountRiskFactorEntries(row); risk != nil {
				riskByRef[account.AccountRef] = risk
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: stream every native account record from the datastore in a single
	// prefix-filtered firehose and fold its native attributes into the same
	// per-account accumulator. No account-ref join (no FK); merkle reconciliation
	// owns change/delete detection.
	if err := collectAccountAttributes(ctx, client, attrs); err != nil {
		return err
	}

	// Emit one consolidated AccountExtension per account seen in either pass.
	for _, ref := range attrs.refs() {
		ext := &entities.AccountExtension{
			Metadata:        types.EntityMetadata{Space: spaces.AccountExtensions},
			AccountRef:      ref,
			Attributes:      attrs.attributesFor(ref),
			Classifications: classByRef[ref],
			RiskFactors:     riskByRef[ref],
		}
		if err := c.Emit(ctx, ext); err != nil {
			return fmt.Errorf("emit account extension %s: %w", ref, err)
		}
	}

	return nil
}
