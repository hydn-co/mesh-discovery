package collectors

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/options"
)

type fakeMesh struct {
	derivedProviders []uuid.UUID
	connectors       []uuid.UUID
	features         []uuid.UUID
	bakedOptions     []json.RawMessage
	secretIDs        []*uuid.UUID
	invocations      int
}

func (f *fakeMesh) DeriveProvider(_ context.Context, _, providerID uuid.UUID, _ api.DeriveProviderRequest) error {
	f.derivedProviders = append(f.derivedProviders, providerID)
	return nil
}

func (f *fakeMesh) PutConnector(_ context.Context, _, connectorID uuid.UUID, _ api.PutConnectorRequest) error {
	f.connectors = append(f.connectors, connectorID)
	return nil
}

func (f *fakeMesh) PutConnectorFeature(
	_ context.Context, _, _, featureID uuid.UUID, req api.PutConnectorFeatureRequest,
) error {
	f.features = append(f.features, featureID)
	f.bakedOptions = append(f.bakedOptions, req.Options)
	f.secretIDs = append(f.secretIDs, req.SecretID)
	return nil
}

func (f *fakeMesh) RequestInvocation(_ context.Context, _, _, _ uuid.UUID, _ int) error {
	f.invocations++
	return nil
}

func TestShouldDeriveProviderPerDatasourceWhenOrchestratorRuns(t *testing.T) {
	secret := uuid.New()
	mesh := &fakeMesh{}
	emitter := &captureEntityEmitter{}
	c := &DiscoverDatasourcesCollector{
		TypedFeatureContext: newContractContext(t, emitter, &options.DiscoverDatasourcesOptions{
			DiscoveryOptionsCore: options.DiscoveryOptionsCore{BaseURL: "https://discovery.example"},
			MeshSecretID:         secret.String(),
		}),
		newClient: fakeFactory,
		newMesh:   func() (meshManager, error) { return mesh, nil },
	}

	runCollector(t, c)

	// fakeDiscoveryClient yields accounts in two datasources: ds1 and ds2.
	require.Len(t, mesh.derivedProviders, 2, "one derived provider per datasource")
	require.Len(t, mesh.connectors, 2, "one connector per datasource")
	require.Len(t, mesh.features, 2*len(perDatasourceFeatures), "three collector features per datasource")
	require.Equal(t, 2*len(perDatasourceFeatures), mesh.invocations)

	// Every connector feature carries the configured discovery secret.
	for _, sid := range mesh.secretIDs {
		require.NotNil(t, sid)
		require.Equal(t, secret, *sid)
	}

	// Baked options decode and carry a datasource scope.
	seenScopes := map[string]bool{}
	for _, raw := range mesh.bakedOptions {
		var opts options.AccountEntityCollectorOptions
		require.NoError(t, json.Unmarshal(raw, &opts))
		require.Equal(t, "https://discovery.example", opts.BaseURL)
		require.NotEmpty(t, opts.DataSourceID)
		seenScopes[opts.DataSourceID] = true
	}
	require.True(t, seenScopes["ds1"])
	require.True(t, seenScopes["ds2"])

	// Derived provider ids are deterministic (stable across runs for idempotency).
	require.Equal(t, uuid.NewSHA1(derivedProviderNamespace, []byte("mesh-discovery-ds1")), mesh.derivedProviders[0])
}
