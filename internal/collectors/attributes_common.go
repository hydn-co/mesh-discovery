package collectors

import (
	"context"
	"fmt"

	"github.com/hydn-co/mesh-sdk/pkg/connector"

	"github.com/hydn-co/mesh-discovery/internal/mappings"
)

// emitNamedAttributes emits one value edge per name/value pair produced by
// mkEdge (the per-entity AccountAttribute / GroupAttribute / PersonAttribute).
// mkEdge must not return a nil pointer for the names passed here.
//
// When seen is non-nil it also emits an Attribute definition the first time each
// name is observed (deduped across an entity's grid + fetched-record passes).
// The shared "attributes" definition space can have only one owning collector
// (otherwise the collectors would merkle-prune each other's dictionary), so the
// account collector passes a set and owns the dictionary while the group and
// owner collectors pass nil and emit value edges only.
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
