package collectors

import (
	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"

	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// attrAccumulator gathers attribute entries per parent ref across a collector's
// multiple passes (search grid + datastore firehose), so each parent can be
// emitted as a single consolidated extension entity instead of one edge per
// attribute (hydn-co/control#1436).
//
// It preserves the first-seen order of parent refs (deterministic emission) and
// dedupes attribute entries by key within a parent — last write wins, matching
// the catalog's compound-ref overwrite semantics where a later pass would have
// overwritten the earlier edge. Holding one entry-set per parent in memory is
// the inherent cost of consolidating a multi-source stream into one entity.
type attrAccumulator struct {
	order []string                             // parent refs, first-seen order
	attrs map[string][]entities.AttributeEntry // parent ref -> entries
	index map[string]map[string]int            // parent ref -> attr ref -> index in attrs[ref]
}

func newAttrAccumulator() *attrAccumulator {
	return &attrAccumulator{
		attrs: make(map[string][]entities.AttributeEntry),
		index: make(map[string]map[string]int),
	}
}

// touch registers a parent ref so it yields an extension even when it has no
// attributes (e.g. an account whose only signal is classifications/risk
// factors, or that has no datastore record).
func (a *attrAccumulator) touch(ref string) {
	if ref == "" {
		return
	}
	if _, ok := a.index[ref]; !ok {
		a.index[ref] = make(map[string]int)
		a.order = append(a.order, ref)
	}
}

// add merges a flattened name->value attribute map onto a parent ref.
func (a *attrAccumulator) add(ref string, named map[string]string) {
	if ref == "" {
		return
	}
	a.touch(ref)
	for _, entry := range mappings.AttributeEntriesFromMap(named) {
		if i, ok := a.index[ref][entry.Ref]; ok {
			a.attrs[ref][i] = entry // last write wins
			continue
		}
		a.index[ref][entry.Ref] = len(a.attrs[ref])
		a.attrs[ref] = append(a.attrs[ref], entry)
	}
}

// refs returns the parent refs in first-seen order.
func (a *attrAccumulator) refs() []string { return a.order }

// attributesFor returns the merged attribute entries for a parent ref.
func (a *attrAccumulator) attributesFor(ref string) []entities.AttributeEntry {
	return a.attrs[ref]
}
