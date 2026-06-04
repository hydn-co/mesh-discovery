// Package credentials extracts the discovery API client credentials from the
// standard mesh Grant credential ({"client_id","client_secret"}).
package credentials

import (
	"encoding/json"

	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
)

// ExtractClientCredentials returns the discovery client id and secret from a
// feature's credential map, reading the default Grant credential slot.
func ExtractClientCredentials(creds map[string]json.RawMessage) (clientID, clientSecret string, err error) {
	return connectorutil.ExtractGrantCredentialFrom(creds, connectorutil.DefaultCredentialName)
}
