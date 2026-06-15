package mappings

import (
	"sort"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"

	"github.com/hydn-co/mesh-discovery/internal/api"
)

// This file builds the inline blob entries carried by the consolidated extension
// entities (AccountExtension/GroupExtension/PeopleExtension). It reuses the
// existing per-edge mappers and projects their (definition, edge) pairs into the
// flat entry structs, so the consolidated entity carries everything the old
// per-edge fan-out did (hydn-co/control#1436).

// AttributeEntriesFromMap converts a flattened name->value attribute map into
// AttributeEntry blob entries, sorted by key for deterministic output. The
// discovery field name is both the stable ref and the human-readable name;
// discovery carries no type hint.
func AttributeEntriesFromMap(attrs map[string]string) []entities.AttributeEntry {
	if len(attrs) == 0 {
		return nil
	}

	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]entities.AttributeEntry, 0, len(names))
	for _, name := range names {
		out = append(out, entities.AttributeEntry{
			Ref:   name,
			Name:  name,
			Value: attrs[name],
		})
	}
	return out
}

// AccountClassificationEntries projects an account row's classifications into
// ClassificationEntry blob entries (ref/name/confidence).
func AccountClassificationEntries(row api.Row) []entities.ClassificationEntry {
	defs, edges := AccountClassifications(row)
	if len(defs) == 0 {
		return nil
	}

	out := make([]entities.ClassificationEntry, 0, len(defs))
	for i := range defs {
		out = append(out, entities.ClassificationEntry{
			Ref:        defs[i].ClassificationRef,
			Name:       defs[i].Name,
			Confidence: edges[i].Confidence,
		})
	}
	return out
}

// AccountRiskFactorEntries projects an account row's risk factors into
// RiskFactorEntry blob entries (ref/name/category/weight/confidence).
func AccountRiskFactorEntries(row api.Row) []entities.RiskFactorEntry {
	defs, edges := AccountRiskFactors(row)
	if len(defs) == 0 {
		return nil
	}

	out := make([]entities.RiskFactorEntry, 0, len(defs))
	for i := range defs {
		out = append(out, entities.RiskFactorEntry{
			Ref:        defs[i].RiskFactorRef,
			Name:       defs[i].Name,
			Category:   defs[i].Category,
			Weight:     defs[i].Weight,
			Confidence: edges[i].Confidence,
		})
	}
	return out
}
