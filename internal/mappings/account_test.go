package mappings

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

func TestShouldMapTypedAccountFields(t *testing.T) {
	row := api.Row{
		"Id":            "acc-1",
		"Account Name":  "Alice",
		"Display Name":  "Alice A.",
		"UPN":           "alice@corp.example",
		"Account Type":  "Service Account",
		"Status":        "Disabled",
		"Email":         "alice@corp.example",
		"Created":       "1700000000000",
		"Last Logon":    "1710000000000",
		"Disabled Time": "1720000000000",
	}

	account := MapAccount(row)
	require.NotNil(t, account)

	assert.Equal(t, "acc-1", account.AccountRef)
	assert.Equal(t, spaces.Accounts, account.Metadata.Space)
	assert.Equal(t, "Alice", account.Name)
	assert.Equal(t, "Alice A.", account.DisplayName)
	assert.Equal(t, "alice@corp.example", account.UPN)
	assert.Equal(t, types.AccountTypeServicePrincipal, account.AccountType)
	assert.False(t, account.Enabled)
	assert.Equal(t, types.AccountStatusDisabled, account.Status)
	require.NotNil(t, account.PrimaryEmail)
	assert.Equal(t, "alice@corp.example", account.PrimaryEmail.Address)
	require.NotNil(t, account.CreatedAt)
	require.NotNil(t, account.LastLoginDate)
	require.NotNil(t, account.DisabledDate)
}

func TestShouldFallBackToEmailForAccountRef(t *testing.T) {
	row := api.Row{"Email": "bob@corp.example", "Account Name": "Bob"}

	account := MapAccount(row)
	require.NotNil(t, account)
	assert.Equal(t, "bob@corp.example", account.AccountRef)
	assert.Equal(t, "bob@corp.example", AccountRef(row))
}

func TestShouldUseAccountRefAsNameWhenNameMissing(t *testing.T) {
	account := MapAccount(api.Row{"Id": "acc-9"})
	require.NotNil(t, account)
	assert.Equal(t, "acc-9", account.Name)
}

func TestShouldReturnNilWhenAccountRowHasNoIdentifier(t *testing.T) {
	assert.Nil(t, MapAccount(api.Row{"Account Name": "Nameless"}))
	assert.Empty(t, AccountRef(api.Row{"Account Name": "Nameless"}))
}

func TestShouldMapAccountStatus(t *testing.T) {
	cases := map[string]types.AccountStatus{
		"":            "",
		"Enabled":     types.AccountStatusEnabled,
		"active":      types.AccountStatusEnabled,
		"Disabled":    types.AccountStatusDisabled,
		"deactivated": types.AccountStatusDisabled,
		"Locked":      types.AccountStatusOther,
	}
	for in, want := range cases {
		assert.Equalf(t, want, mapAccountStatus(in), "status %q", in)
	}
}

func TestShouldMapAccountType(t *testing.T) {
	cases := map[string]types.AccountType{
		"":                types.AccountTypeUser,
		"User":            types.AccountTypeUser,
		"Service Account": types.AccountTypeServicePrincipal,
		"Guest":           types.AccountTypeGuest,
		"root":            types.AccountTypeRoot,
	}
	for in, want := range cases {
		assert.Equalf(t, want, mapAccountType(in), "type %q", in)
	}
}

func TestShouldTreatBlankStatusAsEnabled(t *testing.T) {
	account := MapAccount(api.Row{"Id": "acc-1"})
	require.NotNil(t, account)
	assert.True(t, account.Enabled)
	assert.Equal(t, types.AccountStatus(""), account.Status)
}
