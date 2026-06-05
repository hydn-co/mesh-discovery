// Package credentials extracts the discovery API client credentials from the
// standard mesh Grant credential ({"client_id","client_secret"}).
package credentials

import (
	"encoding/json"

	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
)

// ExtractClientCredentials returns the discovery client id and secret from the
// feature's default mesh Grant credential slot. mesh-sdk v0.2.71+ keys
// credentials by slot name; a single-credential feature resolves under
// connectorutil.DefaultCredentialName.
func ExtractClientCredentials(creds map[string]json.RawMessage) (clientID, clientSecret string, err error) {
	return connectorutil.ExtractGrantCredentialFrom(creds, connectorutil.DefaultCredentialName)
}
