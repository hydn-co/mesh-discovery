// Package credentials extracts the discovery API client credentials from the
// standard mesh Grant credential ({"client_id","client_secret"}).
package credentials

import (
	"encoding/json"

	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
)

// ExtractClientCredentials returns the discovery client id and secret from a
// mesh Grant credential.
func ExtractClientCredentials(raw json.RawMessage) (clientID, clientSecret string, err error) {
	return connectorutil.ExtractGrantCredential(raw)
}
