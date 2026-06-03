package collectors

import (
	"context"

	"github.com/hydn-co/mesh-discovery/internal/api"
	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// collectDatasources scans the account feed for distinct datasources, keyed by
// datasource id and preserving first-seen order. This mirrors control's hydden
// DiscoverApplications: the account feed is the canonical source of the
// (id, name, platform) tuples, since group/role feeds carry only the name.
func collectDatasources(ctx context.Context, client discoveryClient) ([]mappings.Datasource, error) {
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

// datasourceIDByName builds a datasource-name -> datasource-id index from the
// account feed. Group and role rows carry the datasource name (not id), so this
// resolves their application link, matching how control resolves ApplicationID
// by matching DataSourceName against the applications repository.
func datasourceIDByName(ctx context.Context, client discoveryClient) (map[string]string, error) {
	datasources, err := collectDatasources(ctx, client)
	if err != nil {
		return nil, err
	}
	idx := make(map[string]string, len(datasources))
	for _, ds := range datasources {
		if ds.Name != "" {
			idx[ds.Name] = ds.ID
		}
	}
	return idx, nil
}
