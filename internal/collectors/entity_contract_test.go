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
	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
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

// fakeDiscoveryClient returns canned discovery rows across two datasources:
// ds1 ("ds1-name", platform AD) and ds2 ("ds2-name", platform Entra).
type fakeDiscoveryClient struct{}

func (fakeDiscoveryClient) ForEachAccountPage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{
			"Id": "acc-1", "Account Name": "Alice", "Email": "alice@ds1.example",
			"Account Type": "User", "Data Source Id": "ds1",
			"Data Source Name": "ds1-name", "Data Source Platform": "AD",
			// Drives a risk factor, a classification, and a grid attribute.
			"Accounts with MFA Not Enabled": "5", "Classifications": "Admin",
			"Department": "IT",
		},
		{
			"Id": "acc-2", "Account Name": "svc", "Account Type": "Service Account",
			"Data Source Id": "ds1", "Data Source Name": "ds1-name", "Data Source Platform": "AD",
		},
		{
			"Id": "acc-3", "Account Name": "Bob", "Email": "bob@ds2.example",
			"Data Source Id": "ds2", "Data Source Name": "ds2-name", "Data Source Platform": "Entra",
		},
	}, 1, 3)
}

func (fakeDiscoveryClient) ForEachGroupPage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{"Group Id": "grp-1", "Group Name": "Admins", "Data Source Name": "ds1-name", "Group Domain": "corp"},
		{"Group Id": "grp-2", "Group Name": "Others", "Data Source Name": "ds2-name", "Group Domain": "corp"},
	}, 1, 2)
}

func (fakeDiscoveryClient) ForEachOwnerPage(_ context.Context, cb api.PageFunc) error {
	return cb([]api.Row{
		{"Identity Id": "own-1", "Identity Name": "Owner One", "Identity Email": "owner1@example", "Department": "IT"},
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
	switch entityType {
	case "edge.role":
		// edge.role: From = role external id, To = account external id.
		return cb(&api.FetchedEntity{Type: "edge.role", From: "role-1", To: "acc-1"})
	case membershipEntityType:
		// edge.membership: From = group external id, To = member account id.
		return cb(&api.FetchedEntity{Type: membershipEntityType, From: "grp-1", To: "acc-1"})
	case accountEntityTypePrefix:
		return cb(&api.FetchedEntity{
			ID:     "acc-1",
			Type:   "principal.account.user.generic",
			Entity: map[string]any{"department": "IT", "title": "Admin"},
		})
	case groupEntityTypePrefix:
		return cb(&api.FetchedEntity{
			ID:     "grp-1",
			Type:   "group.generic",
			Entity: map[string]any{"distinguished_name": "CN=Admins"},
		})
	case personEntityTypePrefix:
		return cb(&api.FetchedEntity{
			ID:     "own-1",
			Type:   "identity.auto",
			Entity: map[string]any{"manager": "Jane"},
		})
	default:
		return nil
	}
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
				Credentials: map[string]json.RawMessage{connectorutil.DefaultCredentialName: creds},
			}),
			connector.WithEmitter(emitter),
		),
	)
}

func discoveryCore() options.DiscoveryOptionsCore {
	return options.DiscoveryOptionsCore{BaseURL: "https://discovery.example"}
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

func TestShouldEmitApplicationPerDatasourceWhenApplicationCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &ApplicationEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.ApplicationEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted, []any{&entities.Application{}},
		(&options.ApplicationEntityCollectorOptions{}).GetSpaces())

	byRef := map[string]*entities.Application{}
	for _, e := range emitter.emitted {
		app := e.(*entities.Application)
		byRef[app.ApplicationRef] = app
	}
	require.Len(t, byRef, 2)
	require.Equal(t, "ds1-name", byRef["ds1"].Name)
	require.Equal(t, "AD", byRef["ds1"].Description)
	require.Equal(t, "Entra", byRef["ds2"].Description)
}

func TestShouldEmitAccountsAndApplicationLinksWhenAccountCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &AccountEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.AccountEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted, []any{
		&entities.Account{},
		&entities.ApplicationAccount{},
		&entities.AccountExtension{},
	}, (&options.AccountEntityCollectorOptions{}).GetSpaces())

	links := applicationAccountLinks(emitter.emitted)
	require.Equal(t, "ds1", links["acc-1"])
	require.Equal(t, "ds2", links["acc-3"])

	// Attributes, classifications, and risk factors are consolidated onto a single
	// AccountExtension per account rather than fanned out into edges.
	var extensions int
	for _, e := range emitter.emitted {
		if ext, ok := e.(*entities.AccountExtension); ok {
			extensions++
			if ext.AccountRef == "acc-1" {
				require.NotEmpty(t, ext.Attributes, "acc-1 extension carries its attributes inline")
			}
		}
	}
	require.Positive(t, extensions)
}

func TestShouldEmitGroupsMembersAndApplicationLinksWhenGroupCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &GroupEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.GroupEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted,
		[]any{
			&entities.Group{}, &entities.GroupMember{}, &entities.ApplicationGroup{},
			&entities.GroupExtension{},
		},
		(&options.GroupEntityCollectorOptions{}).GetSpaces())

	for _, e := range emitter.emitted {
		if edge, ok := e.(*entities.ApplicationGroup); ok && edge.GroupRef == "grp-1" {
			require.Equal(t, "ds1", edge.ApplicationRef, "group resolves datasource name->id")
		}
	}
}

func TestShouldEmitPersonsWhenOwnerCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &OwnerEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.OwnerEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted,
		[]any{&entities.Person{}, &entities.PeopleExtension{}},
		(&options.OwnerEntityCollectorOptions{}).GetSpaces())
}

func TestShouldEmitRolesLinksAndMembershipsWhenApplicationRoleCollectorRuns(t *testing.T) {
	emitter := &captureEntityEmitter{}
	c := &ApplicationRoleEntityCollector{
		TypedFeatureContext: newContractContext(t, emitter,
			&options.ApplicationRoleEntityCollectorOptions{DiscoveryOptionsCore: discoveryCore()}),
		newClient: fakeFactory,
	}
	runCollector(t, c)

	assertEmittedEntityContract(t, emitter.emitted,
		[]any{&entities.Role{}, &entities.AccountRole{}, &entities.ApplicationRole{}},
		(&options.ApplicationRoleEntityCollectorOptions{}).GetSpaces())

	for _, e := range emitter.emitted {
		switch v := e.(type) {
		case *entities.ApplicationRole:
			if v.RoleRef == "role-1" {
				require.Equal(t, "ds1", v.ApplicationRef, "role resolves datasource name->id")
			}
		case *entities.AccountRole:
			require.Equal(t, "acc-1", v.AccountRef)
			require.Equal(t, "role-1", v.RoleRef)
		}
	}
}

func applicationAccountLinks(emitted []any) map[string]string {
	out := map[string]string{}
	for _, e := range emitted {
		if edge, ok := e.(*entities.ApplicationAccount); ok {
			out[edge.AccountRef] = edge.ApplicationRef
		}
	}
	return out
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

	// The attribute-definition dictionary ("attributes") is additive: collectors
	// emit it but do not declare it as owned. Exempt it from the declared==emitted
	// check; declared (owned) spaces must still match the remaining emitted spaces.
	nonAdditive := make([]spaces.Space, 0, len(observedSpaces))
	for space := range observedSpaces {
		if _, additive := additiveContractSpaces[space]; additive {
			continue
		}
		nonAdditive = append(nonAdditive, space)
	}
	require.ElementsMatch(t, expectedSpaces, nonAdditive)
}

// additiveContractSpaces are emitted-but-not-declared dictionary spaces (see
// mesh-core internal/catalog/spaces.IsAdditive).
var additiveContractSpaces = map[spaces.Space]struct{}{spaces.Attributes: {}}

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
