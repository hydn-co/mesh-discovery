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

	// Pass 1: emit Persons from the owner search grid and accumulate each person's
	// grid-enriched attributes. The consolidated PeopleExtension is emitted after
	// Pass 2 so it carries attributes from both the grid and the datastore
	// (hydn-co/control#1436).
	attrs := newAttrAccumulator()
	if err := client.ForEachOwnerPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			person := mappings.MapOwner(row)
			if person == nil {
				continue
			}
			if err := c.Emit(ctx, person); err != nil {
				return fmt.Errorf("emit person %s: %w", person.PersonRef, err)
			}
			attrs.add(person.PersonRef, mappings.PersonGSAttributes(row))
		}
		return nil
	}); err != nil {
		return err
	}

	// Pass 2: stream every native identity record from the datastore in a single
	// prefix-filtered firehose and fold its native attributes into the per-person
	// accumulator.
	if err := collectPersonAttributes(ctx, client, attrs); err != nil {
		return err
	}

	// Emit one consolidated PeopleExtension per person seen in either pass.
	for _, ref := range attrs.refs() {
		ext := &entities.PeopleExtension{
			Metadata:   types.EntityMetadata{Space: spaces.PeopleExtensions},
			PersonRef:  ref,
			Attributes: attrs.attributesFor(ref),
		}
		if err := c.Emit(ctx, ext); err != nil {
			return fmt.Errorf("emit people extension %s: %w", ref, err)
		}
	}

	return nil
}
