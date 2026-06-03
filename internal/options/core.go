package options

// requirement is the capability tag every discovery feature declares.
const requirement = "discovery"

// DiscoveryOptionsCore holds configuration shared by every discovery feature:
// how to reach the Hydden discovery API. Credentials arrive separately via the
// standard mesh Grant credential.
type DiscoveryOptionsCore struct {
	BaseURL string `json:"base_url" title:"Discovery Base URL" description:"Base URL of the Hydden discovery API (e.g. https://discovery.example.com)." binding:"required"`
}

// GetBaseURL returns the configured discovery base URL (nil-safe).
func (o *DiscoveryOptionsCore) GetBaseURL() string {
	if o == nil {
		return ""
	}
	return o.BaseURL
}

// DatasourceScope scopes a collector to a single discovered datasource. The
// orchestrator bakes these values into each per-datasource connector; when
// DataSourceID is empty the collector emits every datasource's rows (used by
// the base connector before fan-out).
type DatasourceScope struct {
	DataSourceID   string `json:"data_source_id,omitempty"   title:"Data Source Id"   description:"Restrict account/role-membership collection to a single discovery datasource id; empty collects all datasources."`
	DataSourceName string `json:"data_source_name,omitempty" title:"Data Source Name" description:"Restrict group/role collection to a single discovery datasource name (group and role rows are keyed by name, not id)."`
	Platform       string `json:"platform,omitempty"         title:"Platform"         description:"Informational discovered platform name for the scoped datasource."`
}

// GetDataSourceID returns the scoped datasource id (nil-safe).
func (s *DatasourceScope) GetDataSourceID() string {
	if s == nil {
		return ""
	}
	return s.DataSourceID
}

// GetDataSourceName returns the scoped datasource name (nil-safe).
func (s *DatasourceScope) GetDataSourceName() string {
	if s == nil {
		return ""
	}
	return s.DataSourceName
}
