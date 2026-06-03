package collectors

import (
	"context"
	"log/slog"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// discoveryClient is the subset of the Hydden API used by the collectors. It is
// an interface so tests can inject a fake via the newClient seam.
type discoveryClient interface {
	ForEachAccountPage(ctx context.Context, cb api.PageFunc) error
	ForEachGroupPage(ctx context.Context, cb api.PageFunc) error
	ForEachGroupMembershipPage(ctx context.Context, cb api.PageFunc) error
	ForEachOwnerPage(ctx context.Context, cb api.PageFunc) error
	ForEachApplicationRolePage(ctx context.Context, cb api.PageFunc) error
	GetAccountAppRoles(ctx context.Context, accountExternalID string) ([]api.Row, error)
}

// clientFactory builds a discoveryClient; swapped in tests via the collector's
// newClient field.
type clientFactory func(baseURL, clientID, clientSecret string) discoveryClient

func defaultClientFactory(baseURL, clientID, clientSecret string) discoveryClient {
	return api.NewClient(baseURL, clientID, clientSecret, slog.Default())
}
