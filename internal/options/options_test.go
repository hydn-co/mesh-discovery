package options

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/testkit"
)

func TestShouldRegisterPolymorphicOptionsWhenTestKitValidatesRegistrations(t *testing.T) {
	testkit.TestPolymorphicRegistrations(t, map[string]any{
		"mesh://discovery/collectors/account_entity_collector_options":          &AccountEntityCollectorOptions{},
		"mesh://discovery/collectors/group_entity_collector_options":            &GroupEntityCollectorOptions{},
		"mesh://discovery/collectors/owner_entity_collector_options":            &OwnerEntityCollectorOptions{},
		"mesh://discovery/collectors/application_role_entity_collector_options": &ApplicationRoleEntityCollectorOptions{},
	})
}
