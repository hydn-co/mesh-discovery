package collectors

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fgrzl/json/polymorphic"
	"github.com/google/uuid"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"
	"github.com/hydn-co/mesh-sdk/pkg/connector"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/options"
)

// captureEntityEmitter records every emitted entity for assertions.
type captureEntityEmitter struct {
	emitted []any
}

func (e *captureEntityEmitter) Emit(_ context.Context, entity any) error {
	e.emitted = append(e.emitted, entity)
	return nil
}

// fakeDiscoveryClient returns canned discovery rows. It covers two datasources
// ("ds1" / "ds2", names "ds1-name" / "ds2-name") so scoping can be verified.
type fakeDiscoveryClient struct{}

func (fakeDiscoveryClient) ForEachAccountPage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{
			"Id":             "acc-1",
			"Account Name":   "Alice",
			"Email":          "alice@ds1.example",
			"Account Type":   "User",
			"Data Source Id": "ds1",
		},
		{"Id": "acc-2", "Account Name": "svc", "Account Type": "Service Account", "Data Source Id": "ds1"},
		{"Id": "acc-3", "Account Name": "Bob", "Email": "bob@ds2.example", "Data Source Id": "ds2"},
	}, 1, 3)
}

func (fakeDiscoveryClient) ForEachGroupPage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{"Group Id": "grp-1", "Group Name": "Admins", "Data Source Name": "ds1-name"},
		{"Group Id": "grp-2", "Group Name": "Others", "Data Source Name": "ds2-name"},
	}, 1, 2)
}

func (fakeDiscoveryClient) ForEachGroupMembershipPage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{"Group ID": "grp-1", "Account ID": "acc-1"},
		{"Group ID": "grp-2", "Account ID": "acc-3"},
	}, 1, 2)
}

func (fakeDiscoveryClient) ForEachOwnerPage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{"Identity Id": "own-1", "Identity Name": "Owner One", "Identity Email": "owner1@example"},
	}, 1, 1)
}

func (fakeDiscoveryClient) ForEachApplicationRolePage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{"Id": "role-1", "Name": "Global Administrator", "Data Source Name": "ds1-name"},
		{"Id": "role-2", "Name": "Reader", "Data Source Name": "ds2-name"},
	}, 1, 2)
}

func (fakeDiscoveryClient) FetchEntities(
	_ context.Context, _, entityType string, cb func(*api.FetchedEntity) error,
) error {
	if entityType != "edge.role" {
		return nil
	}
	// edge.role: From = role external id, To = account external id.
	return cb(&api.FetchedEntity{Type: "edge.role", From: "role-1", To: "acc-1"})
}

func newContractContext[T connector.FeatureOptions](
	t *testing.T,
	emitter *captureEntityEmitter,
	featureOptions T,
) *connector.TypedFeatureContext[T, *connector.NoPayload] {
	t.Helper()
	creds, err := json.Marshal(map[string]string{"client_id": "cid", "client_secret": "secret"})
	require.NoError(t, err)
	return connector.NewTypedFeatureContext[T, *connector.NoPayload](
		connector.NewFeatureContext(
			connector.WithConfiguration(&connector.Configuration{
				TenantID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				ConnectorID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
				Options:     polymorphic.NewEnvelope(featureOptions),
				Credentials: creds,
			}),
			connector.WithEmitter(emitter),
		),
	)
}

func fakeFactory(_, _, _ string) discoveryClient { return fakeDiscoveryClient{} }

func runCollector(t *testing.T, c interface {
	Init(context.Context) error
	Start(context.Context) error
	Stop(context.Context) error
},
) {
	t.Helper()
	require.NoError(t, c.Init(context.Background()))
	require.NoError(t, c.Start(context.Background()))
	require.NoError(t, c.Stop(context.Background()))
}

func TestShouldOnlyEmitAccountsScopedToDatasourceWhenAccountCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &AccountEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter, &options.AccountEntityCollectorOptions{
			DiscoveryOptionsCore: options.DiscoveryOptionsCore{BaseURL: "https://discovery.example"},
			DatasourceScope:      options.DatasourceScope{DataSourceID: "ds1"},
		}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted, []any{&entities.Account{}},
		(&options.AccountEntityCollectorOptions{}).GetSpaces())
	require.Len(t, emitter.emitted, 2, "only ds1 accounts should be emitted")
	for _, e := range emitter.emitted {
		require.NotEqual(t, "acc-3", e.(*entities.Account).AccountRef)
	}
}

func TestShouldEmitGroupsAndMembersScopedByNameWhenGroupCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &GroupEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter, &options.GroupEntityCollectorOptions{
			DiscoveryOptionsCore: options.DiscoveryOptionsCore{BaseURL: "https://discovery.example"},
			DatasourceScope:      options.DatasourceScope{DataSourceName: "ds1-name"},
		}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted, []any{&entities.Group{}, &entities.GroupMember{}},
		(&options.GroupEntityCollectorOptions{}).GetSpaces())
	for _, e := range emitter.emitted {
		if m, ok := e.(*entities.GroupMember); ok {
			require.Equal(t, "grp-1", m.GroupRef, "only ds1 group members should be emitted")
		}
	}
}

func TestShouldEmitPersonsWhenOwnerCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &OwnerEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter, &options.OwnerEntityCollectorOptions{
			DiscoveryOptionsCore: options.DiscoveryOptionsCore{BaseURL: "https://discovery.example"},
		}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted, []any{&entities.Person{}},
		(&options.OwnerEntityCollectorOptions{}).GetSpaces())
}

func TestShouldEmitRolesAndAccountRolesScopedWhenApplicationRoleCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &ApplicationRoleEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter, &options.ApplicationRoleEntityCollectorOptions{
			DiscoveryOptionsCore: options.DiscoveryOptionsCore{BaseURL: "https://discovery.example"},
			DatasourceScope:      options.DatasourceScope{DataSourceID: "ds1", DataSourceName: "ds1-name"},
		}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted, []any{&entities.Role{}, &entities.AccountRole{}},
		(&options.ApplicationRoleEntityCollectorOptions{}).GetSpaces())
	for _, e := range emitter.emitted {
		if ar, ok := e.(*entities.AccountRole); ok {
			require.Equal(t, "acc-1", ar.AccountRef)
			require.Equal(t, "role-1", ar.RoleRef)
		}
	}
}

// assertEmittedEntityContract verifies the collector emitted only the allowed
// entity types and exactly the spaces its options declare.
func assertEmittedEntityContract(t *testing.T, emitted, allowedTypes []any, expectedSpaces []spaces.Space) {
	t.Helper()
	allowedNames := map[string]struct{}{}
	allowedList := make([]string, 0, len(allowedTypes))
	for _, item := range allowedTypes {
		n := reflect.TypeOf(item).String()
		allowedNames[n] = struct{}{}
		allowedList = append(allowedList, n)
	}

	require.NotEmpty(t, emitted, "expected at least one emitted entity")

	observedTypes := map[string]struct{}{}
	observedSpaces := map[spaces.Space]struct{}{}
	for _, item := range emitted {
		n := reflect.TypeOf(item).String()
		_, ok := allowedNames[n]
		require.Truef(t, ok, "unexpected emitted entity type %s", n)
		observedTypes[n] = struct{}{}
		space, ok := emittedEntitySpace(item)
		require.Truef(t, ok, "entity %s has no metadata space", n)
		observedSpaces[space] = struct{}{}
	}

	require.ElementsMatch(t, allowedList, keys(observedTypes))
	require.ElementsMatch(t, expectedSpaces, spaceKeys(observedSpaces))
}

func emittedEntitySpace(item any) (spaces.Space, bool) {
	e, ok := item.(interface{ GetMetadata() types.EntityMetadata })
	if !ok {
		return "", false
	}
	return e.GetMetadata().Space, true
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func spaceKeys(m map[spaces.Space]struct{}) []spaces.Space {
	out := make([]spaces.Space, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
