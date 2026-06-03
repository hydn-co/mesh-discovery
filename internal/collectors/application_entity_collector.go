package collectors

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hydn-co/mesh-sdk/pkg/connector"
	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
	"github.com/hydn-co/mesh-sdk/pkg/runner"

	"github.com/hydn-co/mesh-discovery/internal/credentials"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
	"github.com/hydn-co/mesh-discovery/internal/options"
)

// ApplicationEntityCollector emits one catalog Application per discovered
// datasource (ApplicationRef = datasource id, Name = datasource name,
// Description = platform). Accounts, groups, and roles link to these via the
// ApplicationAccount/ApplicationGroup/ApplicationRole edges.
type ApplicationEntityCollector struct {
	*connector.TypedFeatureContext[*options.ApplicationEntityCollectorOptions, *connector.NoPayload]

	newClient clientFactory
}

func NewApplicationEntityCollector(
	ctx *connector.TypedFeatureContext[*options.ApplicationEntityCollectorOptions, *connector.NoPayload],
) runner.Feature {
	return &ApplicationEntityCollector{TypedFeatureContext: ctx, newClient: defaultClientFactory}
}

func (c *ApplicationEntityCollector) Init(_ context.Context) error {
	return connectorutil.Validate(c.GetOptions(), "application collector options")
}

func (c *ApplicationEntityCollector) Stop(_ context.Context) error { return nil }

func (c *ApplicationEntityCollector) Start(ctx context.Context) error {
	connectorutil.LogFeature(ctx, c.TypedFeatureContext, slog.LevelInfo, "Starting discovery application collector")

	clientID, clientSecret, err := credentials.ExtractClientCredentials(c.GetCredentials())
	if err != nil {
		return fmt.Errorf("discovery credentials: %w", err)
	}

	client := c.newClient(c.GetOptions().GetBaseURL(), clientID, clientSecret)
	datasources, err := collectDatasources(ctx, client)
	if err != nil {
		return fmt.Errorf("enumerate datasources: %w", err)
	}

	for _, ds := range datasources {
		application := mappings.MapApplication(ds)
		if application == nil {
			continue
		}
		if err := c.Emit(ctx, application); err != nil {
			return fmt.Errorf("emit application %s: %w", application.ApplicationRef, err)
		}
	}
	return nil
}
