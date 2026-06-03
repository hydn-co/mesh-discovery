// Package mappings converts raw Hydden discovery rows into mesh-sdk catalog
// entities. Field names mirror control's hydden mapping_*.go (the contract
// source of truth) so the discovery API shape stays consistent.
package mappings

import (
	"strconv"
	"time"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// getString safely extracts a string value from a discovery row.
func getString(row api.Row, key string) string {
	if v, ok := row[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// firstNonEmpty returns the first non-empty lookup, supporting schema renames.
func firstNonEmpty(row api.Row, keys ...string) string {
	for _, k := range keys {
		if v := getString(row, k); v != "" {
			return v
		}
	}
	return ""
}

// parseEpochMillis converts an epoch-millisecond string to *time.Time.
func parseEpochMillis(s string) *time.Time {
	if s == "" || s == "0" {
		return nil
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil || ms <= 0 {
		return nil
	}
	t := time.UnixMilli(ms)
	return &t
}

// Datasource identifies the upstream platform/datasource a row belongs to.
type Datasource struct {
	ID       string
	Name     string
	Platform string
}

// DatasourceOf extracts the datasource tuple from a discovery row. Mirrors the
// "Data Source Id/Name/Platform" columns used by control's DiscoverApplications.
func DatasourceOf(row api.Row) Datasource {
	return Datasource{
		ID:       getString(row, "Data Source Id"),
		Name:     getString(row, "Data Source Name"),
		Platform: getString(row, "Data Source Platform"),
	}
}
