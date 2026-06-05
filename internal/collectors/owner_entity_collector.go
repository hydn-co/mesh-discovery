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

// OwnerEntityCollector collects discovery owners (global identities) and emits
// them as catalog Person entities. Owners transcend datasources, so this
// collector is not datasource scoped and is registered on the base
// mesh-discovery provider only.
type OwnerEntityCollector struct {
	*connector.TypedFeatureContext[*options.OwnerEntityCollectorOptions, *connector.NoPayload]

	newClient clientFactory
}

func NewOwnerEntityCollector(
	ctx *connector.TypedFeatureContext[*options.OwnerEntityCollectorOptions, *connector.NoPayload],
) runner.Feature {
	return &OwnerEntityCollector{TypedFeatureContext: ctx, newClient: defaultClientFactory}
}

func (c *OwnerEntityCollector) Init(_ context.Context) error {
	return connectorutil.Validate(c.GetOptions(), "owner collector options")
}

func (c *OwnerEntityCollector) Stop(_ context.Context) error { return nil }

func (c *OwnerEntityCollector) Start(ctx context.Context) error {
	connectorutil.LogFeature(ctx, c.TypedFeatureContext, slog.LevelInfo, "Starting discovery owner collector")

	clientID, clientSecret, err := credentials.ExtractClientCredentials(c.GetCredentials())
	if err != nil {
		return fmt.Errorf("discovery credentials: %w", err)
	}

	client := c.newClient(c.GetOptions().GetBaseURL(), clientID, clientSecret)

	// Owners are global identities, not per-datasource datastore entities, so the
	// owner feed itself carries the full person object. Each owner row is emitted
	// as a Person, and its remaining fields are folded into Attribute definitions
	// + PersonAttribute value edges (same flatten/emit pattern as the account and
	// group attribute passes). seenAttr dedupes the definitions this run emits
	// into the additive "attributes" dictionary.
	seenAttr := make(map[string]struct{})
	return client.ForEachOwnerPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			person := mappings.MapOwner(row)
			if person == nil {
				continue
			}
			if err := c.Emit(ctx, person); err != nil {
				return fmt.Errorf("emit person %s: %w", person.PersonRef, err)
			}
			if err := emitNamedAttributes(
				ctx,
				c,
				mappings.PersonGSAttributes(row),
				seenAttr,
				func(name, value string) any { return mappings.NewPersonAttribute(person.PersonRef, name, value) },
			); err != nil {
				return err
			}
		}
		return nil
	})
}
