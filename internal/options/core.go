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
