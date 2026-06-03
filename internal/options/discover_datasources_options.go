package options

import "github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"

// DiscoverDatasourcesOptions configures the orchestrator feature. It reads the
// discovery datasource inventory and registers a derived
// mesh-discovery-<platform> provider + connector per datasource in mesh-core.
//
// The discovery (Hydden) credential arrives via the standard Grant credential.
// The mesh-core management endpoint + system credential come from the
// environment (MESH_CORE_BASE_URL, MESH_CLIENT_ID, MESH_CLIENT_SECRET), matching
// control's mesh client.
type DiscoverDatasourcesOptions struct {
	DiscoveryOptionsCore `json:",inline"`

	// MeshSecretID is the id of the mesh secret holding the discovery Grant
	// credential; it is attached to each per-datasource connector so the
	// collectors can authenticate to discovery.
	MeshSecretID string `json:"mesh_secret_id,omitempty" title:"Mesh Secret Id" description:"Mesh secret id (discovery Grant credential) attached to each per-datasource connector."`

	// DerivedManifestVersion/Source pin the per-datasource providers to the
	// published mesh-discovery release so they resolve to this same executable.
	DerivedManifestVersion string `json:"derived_manifest_version,omitempty" title:"Derived Manifest Version" description:"mesh-discovery manifest version to pin on derived providers."`
	DerivedManifestSource  string `json:"derived_manifest_source,omitempty"  title:"Derived Manifest Source"  description:"mesh-discovery manifest source ref (e.g. release tag) for derived providers."`
}

func (o *DiscoverDatasourcesOptions) GetDiscriminator() string {
	return "mesh://discovery/collectors/discover_datasources_options"
}

// GetSpaces returns no spaces: the orchestrator performs mesh-core management
// side effects and emits no catalog entities.
func (o *DiscoverDatasourcesOptions) GetSpaces() []spaces.Space {
	return nil
}

func (o *DiscoverDatasourcesOptions) GetRequirements() []string {
	return []string{requirement}
}

// GetMeshSecretID returns the configured mesh secret id (nil-safe).
func (o *DiscoverDatasourcesOptions) GetMeshSecretID() string {
	if o == nil {
		return ""
	}
	return o.MeshSecretID
}
