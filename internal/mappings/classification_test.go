package mappings

import (
	"testing"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

func TestShouldSplitClassificationTags(t *testing.T) {
	defs, edges := AccountClassifications(api.Row{
		"Id":              "acc-1",
		"Classifications": "Admin, Sensitive; Tier 1",
	})

	refs := classificationRefs(defs)
	assert.ElementsMatch(t, []string{"admin", "sensitive", "tier-1"}, refs)
	require.Len(t, edges, 3)
	for _, e := range edges {
		assert.Equal(t, spaces.AccountClassifications, e.Metadata.Space)
		assert.Equal(t, "acc-1", e.AccountRef)
		assert.Equal(t, 1.0, e.Confidence)
	}
	for _, d := range defs {
		assert.Equal(t, spaces.Classifications, d.Metadata.Space)
		assert.NotEmpty(t, d.Name)
	}
}

func TestShouldEmitPrivilegeClassifications(t *testing.T) {
	defs, _ := AccountClassifications(api.Row{
		"Id":              "acc-1",
		"Is Privileged":   "true",
		"Is Global Admin": "1",
		"Is Main Account": "yes",
	})
	assert.ElementsMatch(t, []string{"privileged", "global-admin", "main-account"}, classificationRefs(defs))
}

func TestShouldFallBackToGlobalAdminKey(t *testing.T) {
	defs, _ := AccountClassifications(api.Row{"Id": "acc-1", "Global Admin": "true"})
	assert.Equal(t, []string{"global-admin"}, classificationRefs(defs))
}

func TestShouldIgnoreFalseyPrivilegeFlags(t *testing.T) {
	defs, edges := AccountClassifications(api.Row{
		"Id":              "acc-1",
		"Is Privileged":   "false",
		"Is Global Admin": "0",
		"Is Main Account": "",
	})
	assert.Empty(t, defs)
	assert.Empty(t, edges)
}

func TestShouldReturnNilClassificationsWhenNoAccountRef(t *testing.T) {
	defs, edges := AccountClassifications(api.Row{"Classifications": "Admin"})
	assert.Nil(t, defs)
	assert.Nil(t, edges)
}

func TestShouldReportTruthy(t *testing.T) {
	for _, s := range []string{"true", "1", "yes", "Y", "enabled", "5"} {
		assert.Truef(t, truthy(s), "%q should be truthy", s)
	}
	for _, s := range []string{"", "false", "0", "no", "n", "  False  "} {
		assert.Falsef(t, truthy(s), "%q should be falsey", s)
	}
}

func TestShouldSlugifyClassificationTags(t *testing.T) {
	assert.Equal(t, "global-admin", classificationSlug("Global Admin"))
	assert.Equal(t, "tier-1", classificationSlug("  Tier 1  "))
	assert.Equal(t, "admin", classificationSlug("Admin!"))
}

func classificationRefs(defs []*entities.Classification) []string {
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.ClassificationRef)
	}
	return out
}
