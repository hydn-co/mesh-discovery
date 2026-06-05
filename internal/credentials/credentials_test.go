package credentials

import (
	"encoding/json"
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/connectorutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldExtractGrantFromDefaultSlot(t *testing.T) {
	creds := map[string]json.RawMessage{
		connectorutil.DefaultCredentialName: json.RawMessage(`{"client_id":"cid","client_secret":"sec"}`),
	}

	id, secret, err := ExtractClientCredentials(creds)
	require.NoError(t, err)
	assert.Equal(t, "cid", id)
	assert.Equal(t, "sec", secret)
}

func TestShouldErrorWhenDefaultSlotMissing(t *testing.T) {
	_, _, err := ExtractClientCredentials(map[string]json.RawMessage{
		"other": json.RawMessage(`{"client_id":"cid","client_secret":"sec"}`),
	})
	assert.Error(t, err)
}

func TestShouldErrorOnNilCredentials(t *testing.T) {
	_, _, err := ExtractClientCredentials(nil)
	assert.Error(t, err)
}
