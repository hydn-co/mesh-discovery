package collectors

import (
	"encoding/json"

	"github.com/hydn-co/mesh-sdk/pkg/catalog/entities"
)

// maxExtensionChunkBytes is the target ceiling for one extension chunk's
// serialized content. It sits well under the Azure Table ~32 KB per-value limit
// so that even once mesh-core adds the server-owned _metadata (tenant/connector/
// segment/distinct IDs, space, discriminator) the stored entity still fits
// (hydn-co/control#1436).
const maxExtensionChunkBytes = 28000

// extensionChunkOverheadBytes is a fixed allowance per chunk for the entity's
// scalar fields (refs, sequence, total_chunks), JSON envelope, and the
// server-added _metadata, charged against the per-chunk budget.
const extensionChunkOverheadBytes = 512

// extensionChunkContent is one chunk's slice of an extension's blobs. A parent's
// full item set is packed across as many chunks as needed to keep each under the
// size ceiling; callers map each chunk onto the concrete extension entity.
type extensionChunkContent struct {
	Attributes      []entities.AttributeEntry
	Classifications []entities.ClassificationEntry
	RiskFactors     []entities.RiskFactorEntry
}

// chunkExtensionContent greedily packs the items (in order, by type) across
// chunks so each chunk's estimated serialized size stays under
// maxExtensionChunkBytes. It always returns at least one chunk — an empty chunk
// when the parent has no items — so every parent yields at least one extension
// entity. An individual item larger than the ceiling is placed alone (best
// effort); it cannot be split further without losing the item.
func chunkExtensionContent(
	attrs []entities.AttributeEntry,
	classifications []entities.ClassificationEntry,
	riskFactors []entities.RiskFactorEntry,
	refOverhead int,
) []extensionChunkContent {
	var chunks []extensionChunkContent

	cur := extensionChunkContent{}
	base := extensionChunkOverheadBytes + refOverhead
	curSize := base
	curHasItems := false

	flush := func() {
		chunks = append(chunks, cur)
		cur = extensionChunkContent{}
		curSize = base
		curHasItems = false
	}
	// fits charges sz to the current chunk, flushing first when a non-empty chunk
	// would overflow, so each item lands in a chunk that stays under the ceiling.
	fits := func(sz int) {
		if curHasItems && curSize+sz > maxExtensionChunkBytes {
			flush()
		}
		curSize += sz
		curHasItems = true
	}

	for _, a := range attrs {
		fits(jsonSize(a))
		cur.Attributes = append(cur.Attributes, a)
	}
	for _, c := range classifications {
		fits(jsonSize(c))
		cur.Classifications = append(cur.Classifications, c)
	}
	for _, r := range riskFactors {
		fits(jsonSize(r))
		cur.RiskFactors = append(cur.RiskFactors, r)
	}

	if curHasItems || len(chunks) == 0 {
		chunks = append(chunks, cur)
	}

	return chunks
}

// jsonSize estimates the serialized cost of one blob entry, including a trailing
// comma/separator.
func jsonSize(v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return len(b) + 1
}
