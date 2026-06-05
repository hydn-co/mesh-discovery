package collectors

import (
	"context"
	"fmt"

	"github.com/hydn-co/mesh-sdk/pkg/connector"

	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// emitNamedAttributes emits, for each name/value pair, the named Attribute
// definition (the dictionary entry) and the per-entity value edge produced by
// mkEdge (AccountAttribute / GroupAttribute / PersonAttribute). mkEdge must not
// return a nil pointer for the names passed here.
//
// seen dedupes the Attribute definitions within a single collector run (the same
// name recurs across entities and across an entity's grid + fetched-record
// passes). All three collectors emit definitions: "attributes" is an additive
// dictionary space — it is never declared as owned and never pruned by
// reconcile, so multiple collectors emitting the same definition is safe and
// idempotent (identical ref + content hash). Pass nil to emit value edges only.
func emitNamedAttributes(
	ctx context.Context,
	emitter connector.EntityEmitter,
	attrs map[string]string,
	seen map[string]struct{},
	mkEdge func(name, value string) any,
) error {
	for name, value := range attrs {
		if seen != nil {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				if def := mappings.NewAttribute(name); def != nil {
					if err := emitter.Emit(ctx, def); err != nil {
						return fmt.Errorf("emit attribute %q: %w", name, err)
					}
				}
			}
		}
		if edge := mkEdge(name, value); edge != nil {
			if err := emitter.Emit(ctx, edge); err != nil {
				return fmt.Errorf("emit attribute value %q: %w", name, err)
			}
		}
	}
	return nil
}
