package mappings

import (
	"strings"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// AccountRef returns the catalog reference for an account row: the Hydden "Id"
// with an "Email" fallback. Used so AccountRole/GroupMember refs line up with
// the Account entities the account collector emits.
func AccountRef(row api.Row) string {
	if id := getString(row, "Id"); id != "" {
		return id
	}
	return getString(row, "Email")
}

// MapAccount converts a Hydden account row into a catalog Account. Returns nil
// when the row has no usable identifier. Field names match control's
// mapping_account.go ("Id", "Account Name", "Display Name", "Email", ...).
func MapAccount(row api.Row) *entities.Account {
	accountRef := AccountRef(row)
	if accountRef == "" {
		return nil
	}

	name := firstNonEmpty(row, "Account Name", "Display Name")
	if name == "" {
		name = accountRef
	}

	account := &entities.Account{
		Metadata:    types.EntityMetadata{Space: spaces.Accounts},
		AccountRef:  accountRef,
		AccountType: mapAccountType(getString(row, "Account Type")),
		Name:        name,
		DisplayName: getString(row, "Display Name"),
		Enabled:     accountEnabled(getString(row, "Status")),
		CreatedAt:   parseEpochMillis(getString(row, "Created")),
	}

	if email := getString(row, "Email"); email != "" {
		account.PrimaryEmail = &types.Email{Address: email}
	}
	return account
}

// mapAccountType maps a free-text Hydden account type to the catalog enum.
func mapAccountType(raw string) types.AccountType {
	switch s := strings.ToLower(strings.TrimSpace(raw)); {
	case s == "":
		return types.AccountTypeUser
	case strings.Contains(s, "service"), strings.Contains(s, "principal"), strings.Contains(s, "app"):
		return types.AccountTypeServicePrincipal
	case strings.Contains(s, "guest"):
		return types.AccountTypeGuest
	case strings.Contains(s, "group"):
		return types.AccountTypeGroup
	case strings.Contains(s, "root"):
		return types.AccountTypeRoot
	default:
		return types.AccountTypeUser
	}
}

// accountEnabled treats an account as enabled unless the status explicitly
// indicates otherwise. control's catalog push currently emits Enabled=true for
// all accounts; this preserves that while honoring an explicit disabled status.
func accountEnabled(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "disabled", "inactive", "deactivated", "suspended", "deleted":
		return false
	default:
		return true
	}
}
