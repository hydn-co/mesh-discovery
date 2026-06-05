package mappings

import (
	"regexp"
	"strings"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/spaces"
	"github.com/hydn-co/mesh-sdk/pkg/catalog/types"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// privilegeClassification maps a boolean privilege grid column to a
// classification applied when the column is truthy. keys are tried in order so
// schema renames (e.g. "Is Global Admin" vs "Global Admin") still resolve.
type privilegeClassification struct {
	keys []string
	ref  string
	name string
}

var privilegeClassifications = []privilegeClassification{
	{[]string{"Is Privileged"}, "privileged", "Privileged"},
	{[]string{"Is Global Admin", "Global Admin"}, "global-admin", "Global Admin"},
	{[]string{"Is Main Account"}, "main-account", "Main Account"},
}

// classificationGSKeys is the set of grid columns consumed as classifications,
// so the attribute mapper can skip them.
var classificationGSKeys = func() map[string]struct{} {
	out := map[string]struct{}{"Classifications": {}}
	for _, pc := range privilegeClassifications {
		for _, k := range pc.keys {
			out[k] = struct{}{}
		}
	}
	return out
}()

// AccountClassifications returns the Classification definitions and
// AccountClassification edges implied by an account row: the discovery
// "Classifications" tag list plus the privilege boolean columns. Confidence is
// always 1.0 — classifications are asserted directly by discovery.
func AccountClassifications(row api.Row) ([]*entities.Classification, []*entities.AccountClassification) {
	accountRef := AccountRef(row)
	if accountRef == "" {
		return nil, nil
	}

	var (
		defs  []*entities.Classification
		edges []*entities.AccountClassification
	)
	add := func(ref, name string) {
		if ref == "" {
			return
		}
		defs = append(defs, &entities.Classification{
			Metadata:          types.EntityMetadata{Space: spaces.Classifications},
			ClassificationRef: ref,
			Name:              name,
		})
		edges = append(edges, &entities.AccountClassification{
			Metadata:          types.EntityMetadata{Space: spaces.AccountClassifications},
			AccountRef:        accountRef,
			ClassificationRef: ref,
			Confidence:        1,
		})
	}

	for _, tag := range splitClassifications(getString(row, "Classifications")) {
		add(classificationSlug(tag), tag)
	}
	for _, pc := range privilegeClassifications {
		if truthy(firstNonEmpty(row, pc.keys...)) {
			add(pc.ref, pc.name)
		}
	}
	return defs, edges
}

var classificationSplit = regexp.MustCompile(`[;,]`)

// splitClassifications splits a delimited "Classifications" cell into trimmed,
// non-empty tags.
func splitClassifications(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := classificationSplit.Split(s, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// classificationSlug derives a stable ref from a classification tag.
func classificationSlug(tag string) string {
	return strings.Trim(slugNonAlnum.ReplaceAllString(strings.ToLower(strings.TrimSpace(tag)), "-"), "-")
}

// truthy reports whether a discovery boolean-ish cell indicates true. Empty and
// explicit falsey values are false; any other non-empty value is true (matching
// control's boolIndicator semantics for presence columns).
func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "false", "0", "no", "n":
		return false
	default:
		return true
	}
}
