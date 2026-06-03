package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/hydn-co/mesh-sdk/pkg/connector"
	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
	"github.com/hydn-co/mesh-sdk/pkg/runner"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/credentials"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
	"github.com/hydn-co/mesh-discovery/internal/options"
)

// derivedProviderNamespace seeds deterministic ids for derived providers so
// repeated orchestration runs upsert rather than duplicate.
var derivedProviderNamespace = uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hydn-co/mesh-discovery"))

// perDatasourceFeatures are the collectors registered on each derived
// per-datasource provider (collect_owners is global, so it is not included).
var perDatasourceFeatures = []string{"collect_accounts", "collect_groups", "collect_application_roles"}

// meshManager is the subset of the mesh-core management API the orchestrator
// uses. It is an interface so tests can inject a fake.
type meshManager interface {
	DeriveProvider(ctx context.Context, tenantID, providerID uuid.UUID, req api.DeriveProviderRequest) error
	PutConnector(ctx context.Context, tenantID, connectorID uuid.UUID, req api.PutConnectorRequest) error
	PutConnectorFeature(
		ctx context.Context,
		tenantID, connectorID, featureID uuid.UUID,
		req api.PutConnectorFeatureRequest,
	) error
	RequestInvocation(ctx context.Context, tenantID, connectorID, featureID uuid.UUID, runSequence int) error
}

type meshManagerFactory func() (meshManager, error)

// DiscoverDatasourcesCollector is the orchestrator feature. It enumerates the
// discovery datasource inventory and, for each datasource, registers a derived
// mesh-discovery-<platform> provider + connector in mesh-core (reusing this same
// executable) with the datasource baked into the connector's options.
type DiscoverDatasourcesCollector struct {
	*connector.TypedFeatureContext[*options.DiscoverDatasourcesOptions, *connector.NoPayload]

	newClient clientFactory
	newMesh   meshManagerFactory
}

func NewDiscoverDatasourcesCollector(
	ctx *connector.TypedFeatureContext[*options.DiscoverDatasourcesOptions, *connector.NoPayload],
) runner.Feature {
	return &DiscoverDatasourcesCollector{
		TypedFeatureContext: ctx,
		newClient:           defaultClientFactory,
		newMesh:             defaultMeshManagerFactory,
	}
}

func defaultMeshManagerFactory() (meshManager, error) {
	base := os.Getenv("MESH_CORE_BASE_URL")
	clientID := os.Getenv("MESH_CLIENT_ID")
	clientSecret := os.Getenv("MESH_CLIENT_SECRET")
	if base == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("orchestrator requires MESH_CORE_BASE_URL, MESH_CLIENT_ID, MESH_CLIENT_SECRET")
	}
	return api.NewMeshMgmtClient(base, clientID, clientSecret), nil
}

func (c *DiscoverDatasourcesCollector) Init(_ context.Context) error {
	return connectorutil.Validate(c.GetOptions(), "discover datasources options")
}

func (c *DiscoverDatasourcesCollector) Stop(_ context.Context) error { return nil }

func (c *DiscoverDatasourcesCollector) Start(ctx context.Context) error {
	connectorutil.LogFeature(ctx, c.TypedFeatureContext, slog.LevelInfo, "Starting discovery datasource orchestrator")

	clientID, clientSecret, err := credentials.ExtractClientCredentials(c.GetCredentials())
	if err != nil {
		return fmt.Errorf("discovery credentials: %w", err)
	}

	mesh, err := c.newMesh()
	if err != nil {
		return err
	}

	opts := c.GetOptions()
	client := c.newClient(opts.GetBaseURL(), clientID, clientSecret)

	datasources, err := enumerateDatasources(ctx, client)
	if err != nil {
		return fmt.Errorf("enumerate datasources: %w", err)
	}
	connectorutil.LogFeature(ctx, c.TypedFeatureContext, slog.LevelInfo,
		"Discovered datasources", "count", len(datasources))

	tenantID := c.GetTenantID()
	for _, ds := range datasources {
		if err := c.registerDatasource(ctx, mesh, tenantID, ds); err != nil {
			return fmt.Errorf("register datasource %s: %w", ds.ID, err)
		}
	}
	return nil
}

// enumerateDatasources scans the account feed for distinct datasources, mirroring
// control's hydden DiscoverApplications (sync_service.go).
func enumerateDatasources(ctx context.Context, client discoveryClient) ([]mappings.Datasource, error) {
	seen := make(map[string]mappings.Datasource)
	order := make([]string, 0)
	err := client.ForEachAccountPage(ctx, func(page []api.Row, _, _ int) error {
		for _, row := range page {
			ds := mappings.DatasourceOf(row)
			if ds.ID == "" {
				continue
			}
			if _, ok := seen[ds.ID]; !ok {
				seen[ds.ID] = ds
				order = append(order, ds.ID)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]mappings.Datasource, 0, len(order))
	for _, id := range order {
		out = append(out, seen[id])
	}
	return out, nil
}

// registerDatasource derives a provider + connector for one datasource and bakes
// the datasource scope into each collector feature.
func (c *DiscoverDatasourcesCollector) registerDatasource(
	ctx context.Context,
	mesh meshManager,
	tenantID uuid.UUID,
	ds mappings.Datasource,
) error {
	opts := c.GetOptions()
	providerName := derivedProviderName(ds)
	providerID := uuid.NewSHA1(derivedProviderNamespace, []byte(providerName))
	connectorID := uuid.NewSHA1(providerID, []byte("connector:"+ds.ID))

	// 1. Derive the provider (reuses the mesh-discovery executable).
	if err := mesh.DeriveProvider(ctx, tenantID, providerID, api.DeriveProviderRequest{
		DisplayName:       derivedProviderDisplayName(ds),
		Description:       "Discovery datasource " + ds.Name,
		ExecutableName:    "mesh-discovery",
		ManifestVersion:   opts.DerivedManifestVersion,
		ManifestSourceRef: opts.DerivedManifestSource,
		Features:          derivedFeatureManifest(),
	}); err != nil {
		return err
	}

	// 2. Create the connector instance.
	if err := mesh.PutConnector(ctx, tenantID, connectorID, api.PutConnectorRequest{
		Name:       derivedProviderDisplayName(ds),
		ProviderID: providerID,
	}); err != nil {
		return err
	}

	// 3. Bake datasource-scoped options into each collector feature.
	var secretID *uuid.UUID
	if raw := opts.GetMeshSecretID(); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			secretID = &parsed
		} else {
			return fmt.Errorf("invalid mesh_secret_id %q: %w", raw, err)
		}
	}
	for _, name := range perDatasourceFeatures {
		featureID := uuid.NewSHA1(providerID, []byte("feature:"+name))
		baked, err := bakedScopeOptions(opts.GetBaseURL(), ds)
		if err != nil {
			return err
		}
		if err := mesh.PutConnectorFeature(ctx, tenantID, connectorID, featureID, api.PutConnectorFeatureRequest{
			Enabled:  true,
			SecretID: secretID,
			Options:  baked,
		}); err != nil {
			return err
		}
		if err := mesh.RequestInvocation(ctx, tenantID, connectorID, featureID, 1); err != nil {
			return err
		}
	}
	return nil
}

// bakedScopeOptions builds the per-datasource collector options payload. It uses
// the same option struct the collectors decode, so the baked values stay schema
// consistent.
func bakedScopeOptions(baseURL string, ds mappings.Datasource) (json.RawMessage, error) {
	return json.Marshal(options.AccountEntityCollectorOptions{
		DiscoveryOptionsCore: options.DiscoveryOptionsCore{BaseURL: baseURL},
		DatasourceScope: options.DatasourceScope{
			DataSourceID:   ds.ID,
			DataSourceName: ds.Name,
			Platform:       ds.Platform,
		},
	})
}

// derivedFeatureManifest describes the collector features for a derived provider.
func derivedFeatureManifest() map[string]api.DeriveProviderFeature {
	schema := json.RawMessage(`{"type":"object"}`)
	features := map[string]api.DeriveProviderFeature{}
	titles := map[string]string{
		"collect_accounts":          "Collect Accounts",
		"collect_groups":            "Collect Groups",
		"collect_application_roles": "Collect Application Roles",
	}
	for _, name := range perDatasourceFeatures {
		features[name] = api.DeriveProviderFeature{
			DisplayName:    titles[name],
			Type:           "collector",
			ResumeBehavior: "none",
			Schedulable:    true,
			OptionsSchema:  schema,
			Requirements:   []string{"discovery"},
		}
	}
	return features
}

func derivedProviderName(ds mappings.Datasource) string {
	key := ds.Platform
	if key == "" {
		key = ds.ID
	}
	return "mesh-discovery-" + key
}

func derivedProviderDisplayName(ds mappings.Datasource) string {
	if ds.Name != "" {
		return "Discovery: " + ds.Name
	}
	return "Discovery: " + ds.ID
}
